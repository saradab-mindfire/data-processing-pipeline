package controllers

import (
	"bytes"
	"encoding/json"
	"fmt"
	"net/http"
	"net/http/httptest"
	"sync"
	"testing"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/glebarez/sqlite"
	"github.com/google/uuid"
	"gorm.io/gorm"

	"github.com/saradab-mindfire/data-processing-pipeline/packages/database"
	"github.com/saradab-mindfire/data-processing-pipeline/packages/models"
	"github.com/saradab-mindfire/data-processing-pipeline/packages/workerclient"
)

func setupTestDB(t *testing.T) *gorm.DB {
	t.Helper()

	dsn := fmt.Sprintf("file:%s?mode=memory&cache=shared", uuid.NewString())
	db, err := gorm.Open(sqlite.Open(dsn), &gorm.Config{})
	if err != nil {
		t.Fatalf("failed to open in-memory database: %v", err)
	}
	if err := db.AutoMigrate(&models.Pipeline{}, &models.PipelineError{}); err != nil {
		t.Fatalf("failed to migrate in-memory database: %v", err)
	}

	previous := database.Instance
	database.Instance = db
	t.Cleanup(func() { database.Instance = previous })

	return db
}

// fakeWorkerConfig configures a fakeWorker before it starts serving.
type fakeWorkerConfig struct {
	enqueueStatus int
	progress      map[string][3]int
}

// fakeWorker is a stand-in for the worker's internal HTTP API.
type fakeWorker struct {
	enqueueStatus int
	progress      map[string][3]int

	mu           sync.Mutex
	cancelledIDs []string
}

func (f *fakeWorker) wasCancelled(id string) bool {
	f.mu.Lock()
	defer f.mu.Unlock()
	for _, got := range f.cancelledIDs {
		if got == id {
			return true
		}
	}
	return false
}

func newFakeWorker(t *testing.T, cfg fakeWorkerConfig) *fakeWorker {
	t.Helper()
	if cfg.enqueueStatus == 0 {
		cfg.enqueueStatus = http.StatusAccepted
	}
	state := &fakeWorker{enqueueStatus: cfg.enqueueStatus, progress: cfg.progress}

	mux := http.NewServeMux()
	mux.HandleFunc("POST /internal/pipelines", func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(state.enqueueStatus)
	})
	mux.HandleFunc("GET /internal/pipelines/{id}/progress", func(w http.ResponseWriter, r *http.Request) {
		id := r.PathValue("id")
		vals, ok := state.progress[id]
		if !ok {
			w.WriteHeader(http.StatusNotFound)
			return
		}
		json.NewEncoder(w).Encode(map[string]int{
			"processed": vals[0], "valid": vals[1], "invalid": vals[2],
		})
	})
	mux.HandleFunc("POST /internal/pipelines/{id}/cancel", func(w http.ResponseWriter, r *http.Request) {
		state.mu.Lock()
		state.cancelledIDs = append(state.cancelledIDs, r.PathValue("id"))
		state.mu.Unlock()
		json.NewEncoder(w).Encode(map[string]bool{"cancelled": true})
	})

	server := httptest.NewServer(mux)
	workerclient.Init(server.URL)
	t.Cleanup(func() {
		server.Close()
		workerclient.Init("")
	})
	return state
}

func newTestRouter() *gin.Engine {
	gin.SetMode(gin.TestMode)
	r := gin.New()
	r.POST("/pipelines", CreatePipeline)
	r.GET("/pipelines", GetPipelines)
	r.GET("/pipelines/:id", GetPipeline)
	r.PUT("/pipelines/:id", UpdatePipeline)
	r.GET("/pipelines/:id/progress", GetPipelineProgress)
	r.GET("/pipelines/:id/results", GetPipelineResults)
	r.GET("/pipelines/:id/errors", GetPipelineErrors)
	r.PATCH("/pipelines/:id/cancel", CancelPipeline)
	r.DELETE("/pipelines/:id", DeletePipeline)
	return r
}

func doRequest(r *gin.Engine, method, path string, body any) *httptest.ResponseRecorder {
	var reader *bytes.Reader
	if body != nil {
		data, _ := json.Marshal(body)
		reader = bytes.NewReader(data)
	} else {
		reader = bytes.NewReader(nil)
	}
	req := httptest.NewRequest(method, path, reader)
	req.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)
	return w
}

func TestCreatePipeline(t *testing.T) {
	db := setupTestDB(t)
	newFakeWorker(t, fakeWorkerConfig{})
	r := newTestRouter()

	t.Run("invalid body", func(t *testing.T) {
		req := httptest.NewRequest(http.MethodPost, "/pipelines", bytes.NewReader([]byte("not json")))
		req.Header.Set("Content-Type", "application/json")
		w := httptest.NewRecorder()
		r.ServeHTTP(w, req)
		if w.Code != http.StatusBadRequest {
			t.Errorf("status = %d, want %d", w.Code, http.StatusBadRequest)
		}
	})

	t.Run("success", func(t *testing.T) {
		w := doRequest(r, http.MethodPost, "/pipelines", models.PipelineRequest{
			Sources:    []models.SourceConfig{{Type: "csv", Path: "a.csv"}},
			ExportType: "json",
		})
		if w.Code != http.StatusAccepted {
			t.Fatalf("status = %d, want %d, body=%s", w.Code, http.StatusAccepted, w.Body.String())
		}

		var created models.Pipeline
		if err := json.Unmarshal(w.Body.Bytes(), &created); err != nil {
			t.Fatalf("failed to decode response: %v", err)
		}
		if created.Status != models.StatusPending {
			t.Errorf("status = %q, want %q", created.Status, models.StatusPending)
		}

		var stored models.Pipeline
		if err := db.First(&stored, "id = ?", created.ID).Error; err != nil {
			t.Errorf("expected pipeline to be persisted: %v", err)
		}
	})

	t.Run("worker enqueue failure", func(t *testing.T) {
		db2 := setupTestDB(t)
		newFakeWorker(t, fakeWorkerConfig{enqueueStatus: http.StatusInternalServerError})
		r2 := newTestRouter()

		w := doRequest(r2, http.MethodPost, "/pipelines", models.PipelineRequest{ExportType: "json"})
		if w.Code != http.StatusInternalServerError {
			t.Errorf("status = %d, want %d", w.Code, http.StatusInternalServerError)
		}
		_ = db2
	})
}

func TestGetPipelines(t *testing.T) {
	db := setupTestDB(t)
	r := newTestRouter()

	w := doRequest(r, http.MethodGet, "/pipelines", nil)
	if w.Code != http.StatusOK {
		t.Fatalf("status = %d, want %d", w.Code, http.StatusOK)
	}
	var empty []models.Pipeline
	if err := json.Unmarshal(w.Body.Bytes(), &empty); err != nil {
		t.Fatalf("failed to decode response: %v", err)
	}
	if len(empty) != 0 {
		t.Errorf("got %d pipelines, want 0", len(empty))
	}

	db.Create(&models.Pipeline{ID: "11111111-1111-1111-1111-111111111111", Name: "p1", Status: models.StatusPending})
	db.Create(&models.Pipeline{ID: "22222222-2222-2222-2222-222222222222", Name: "p2", Status: models.StatusPending})

	w = doRequest(r, http.MethodGet, "/pipelines", nil)
	var list []models.Pipeline
	json.Unmarshal(w.Body.Bytes(), &list)
	if len(list) != 2 {
		t.Errorf("got %d pipelines, want 2", len(list))
	}
}

func TestGetPipeline(t *testing.T) {
	db := setupTestDB(t)
	r := newTestRouter()

	t.Run("invalid id", func(t *testing.T) {
		w := doRequest(r, http.MethodGet, "/pipelines/not-a-uuid", nil)
		if w.Code != http.StatusBadRequest {
			t.Errorf("status = %d, want %d", w.Code, http.StatusBadRequest)
		}
	})

	t.Run("not found", func(t *testing.T) {
		w := doRequest(r, http.MethodGet, "/pipelines/33333333-3333-3333-3333-333333333333", nil)
		if w.Code != http.StatusNotFound {
			t.Errorf("status = %d, want %d", w.Code, http.StatusNotFound)
		}
	})

	t.Run("found", func(t *testing.T) {
		pipeline := models.Pipeline{ID: "44444444-4444-4444-4444-444444444444", Name: "p", Status: models.StatusPending}
		db.Create(&pipeline)

		w := doRequest(r, http.MethodGet, "/pipelines/"+pipeline.ID, nil)
		if w.Code != http.StatusOK {
			t.Fatalf("status = %d, want %d", w.Code, http.StatusOK)
		}
		var got models.Pipeline
		json.Unmarshal(w.Body.Bytes(), &got)
		if got.ID != pipeline.ID {
			t.Errorf("id = %q, want %q", got.ID, pipeline.ID)
		}
	})
}

func TestUpdatePipeline(t *testing.T) {
	db := setupTestDB(t)
	r := newTestRouter()

	t.Run("not found", func(t *testing.T) {
		w := doRequest(r, http.MethodPut, "/pipelines/55555555-5555-5555-5555-555555555555", models.Pipeline{Name: "x"})
		if w.Code != http.StatusNotFound {
			t.Errorf("status = %d, want %d", w.Code, http.StatusNotFound)
		}
	})

	t.Run("updates fields and preserves id", func(t *testing.T) {
		pipeline := models.Pipeline{ID: "66666666-6666-6666-6666-666666666666", Name: "old-name", Status: models.StatusPending}
		db.Create(&pipeline)

		w := doRequest(r, http.MethodPut, "/pipelines/"+pipeline.ID, models.Pipeline{
			ID:   "attempted-id-change",
			Name: "new-name",
		})
		if w.Code != http.StatusOK {
			t.Fatalf("status = %d, want %d, body=%s", w.Code, http.StatusOK, w.Body.String())
		}

		var stored models.Pipeline
		db.First(&stored, "id = ?", pipeline.ID)
		if stored.Name != "new-name" {
			t.Errorf("name = %q, want %q", stored.Name, "new-name")
		}

		var other models.Pipeline
		if err := db.First(&other, "id = ?", "attempted-id-change").Error; err == nil {
			t.Error("update should not have been able to change the pipeline's id")
		}
	})

	t.Run("invalid body", func(t *testing.T) {
		pipeline := models.Pipeline{ID: "77777777-7777-7777-7777-777777777777", Name: "p", Status: models.StatusPending}
		db.Create(&pipeline)

		req := httptest.NewRequest(http.MethodPut, "/pipelines/"+pipeline.ID, bytes.NewReader([]byte("not json")))
		req.Header.Set("Content-Type", "application/json")
		w := httptest.NewRecorder()
		r.ServeHTTP(w, req)
		if w.Code != http.StatusBadRequest {
			t.Errorf("status = %d, want %d", w.Code, http.StatusBadRequest)
		}
	})
}

func TestGetPipelineProgress(t *testing.T) {
	db := setupTestDB(t)
	r := newTestRouter()

	t.Run("not found", func(t *testing.T) {
		w := doRequest(r, http.MethodGet, "/pipelines/88888888-8888-8888-8888-888888888888/progress", nil)
		if w.Code != http.StatusNotFound {
			t.Errorf("status = %d, want %d", w.Code, http.StatusNotFound)
		}
	})

	t.Run("falls back to stored db counters", func(t *testing.T) {
		newFakeWorker(t, fakeWorkerConfig{progress: map[string][3]int{}})
		pipeline := models.Pipeline{
			ID: "99999999-9999-9999-9999-999999999999", Status: models.StatusProcessing,
			TotalRecords: 10, ProcessedRecords: 4, ValidRecords: 3, InvalidRecords: 1,
		}
		db.Create(&pipeline)

		w := doRequest(r, http.MethodGet, "/pipelines/"+pipeline.ID+"/progress", nil)
		if w.Code != http.StatusOK {
			t.Fatalf("status = %d, want %d", w.Code, http.StatusOK)
		}
		var body map[string]any
		json.Unmarshal(w.Body.Bytes(), &body)
		if body["processed_records"].(float64) != 4 {
			t.Errorf("processed_records = %v, want 4", body["processed_records"])
		}
		if body["percentage"].(float64) != 40 {
			t.Errorf("percentage = %v, want 40", body["percentage"])
		}
	})

	t.Run("uses live worker progress when available", func(t *testing.T) {
		pipeline := models.Pipeline{
			ID: "aaaaaaaa-aaaa-aaaa-aaaa-aaaaaaaaaaaa", Status: models.StatusProcessing,
			TotalRecords: 10, ProcessedRecords: 4, ValidRecords: 3, InvalidRecords: 1,
		}
		db.Create(&pipeline)
		newFakeWorker(t, fakeWorkerConfig{progress: map[string][3]int{
			pipeline.ID: {8, 7, 1},
		}})

		w := doRequest(r, http.MethodGet, "/pipelines/"+pipeline.ID+"/progress", nil)
		var body map[string]any
		json.Unmarshal(w.Body.Bytes(), &body)
		if body["processed_records"].(float64) != 8 {
			t.Errorf("processed_records = %v, want 8 (live worker value)", body["processed_records"])
		}
		if body["percentage"].(float64) != 80 {
			t.Errorf("percentage = %v, want 80", body["percentage"])
		}
	})
}

func TestGetPipelineResults(t *testing.T) {
	db := setupTestDB(t)
	newFakeWorker(t, fakeWorkerConfig{})
	r := newTestRouter()

	t.Run("not found", func(t *testing.T) {
		w := doRequest(r, http.MethodGet, "/pipelines/bbbbbbbb-bbbb-bbbb-bbbb-bbbbbbbbbbbb/results", nil)
		if w.Code != http.StatusNotFound {
			t.Errorf("status = %d, want %d", w.Code, http.StatusNotFound)
		}
	})

	t.Run("not completed yet", func(t *testing.T) {
		pipeline := models.Pipeline{ID: "cccccccc-cccc-cccc-cccc-cccccccccccc", Status: models.StatusProcessing}
		db.Create(&pipeline)

		w := doRequest(r, http.MethodGet, "/pipelines/"+pipeline.ID+"/results", nil)
		if w.Code != http.StatusConflict {
			t.Errorf("status = %d, want %d", w.Code, http.StatusConflict)
		}
	})

	t.Run("completed", func(t *testing.T) {
		pipeline := models.Pipeline{
			ID: "dddddddd-dddd-dddd-dddd-dddddddddddd", Status: models.StatusCompleted,
			TotalRecords: 5, ProcessedRecords: 5, ValidRecords: 4, InvalidRecords: 1,
			CompletedAt: time.Now(),
		}
		db.Create(&pipeline)

		w := doRequest(r, http.MethodGet, "/pipelines/"+pipeline.ID+"/results", nil)
		if w.Code != http.StatusOK {
			t.Fatalf("status = %d, want %d", w.Code, http.StatusOK)
		}
		var body map[string]any
		json.Unmarshal(w.Body.Bytes(), &body)
		wantURL := workerclient.ExportBaseURL() + "/exports/" + pipeline.ID + ".json"
		if body["export_url"] != wantURL {
			t.Errorf("export_url = %v, want %q", body["export_url"], wantURL)
		}
	})
}

func TestGetPipelineErrors(t *testing.T) {
	db := setupTestDB(t)
	r := newTestRouter()

	t.Run("not found", func(t *testing.T) {
		w := doRequest(r, http.MethodGet, "/pipelines/eeeeeeee-eeee-eeee-eeee-eeeeeeeeeeee/errors", nil)
		if w.Code != http.StatusNotFound {
			t.Errorf("status = %d, want %d", w.Code, http.StatusNotFound)
		}
	})

	t.Run("returns associated errors", func(t *testing.T) {
		pipeline := models.Pipeline{ID: "ffffffff-ffff-ffff-ffff-ffffffffffff", Status: models.StatusFailed}
		db.Create(&pipeline)
		db.Create(&models.PipelineError{PipelineID: pipeline.ID, Message: "bad record"})
		db.Create(&models.PipelineError{PipelineID: pipeline.ID, Message: "another error"})

		w := doRequest(r, http.MethodGet, "/pipelines/"+pipeline.ID+"/errors", nil)
		if w.Code != http.StatusOK {
			t.Fatalf("status = %d, want %d", w.Code, http.StatusOK)
		}
		var errs []models.PipelineError
		json.Unmarshal(w.Body.Bytes(), &errs)
		if len(errs) != 2 {
			t.Errorf("got %d errors, want 2", len(errs))
		}
	})
}

func TestCancelPipeline(t *testing.T) {
	db := setupTestDB(t)
	fw := newFakeWorker(t, fakeWorkerConfig{})
	r := newTestRouter()

	t.Run("not found", func(t *testing.T) {
		w := doRequest(r, http.MethodPatch, "/pipelines/00000000-0000-0000-0000-000000000000/cancel", nil)
		if w.Code != http.StatusNotFound {
			t.Errorf("status = %d, want %d", w.Code, http.StatusNotFound)
		}
	})

	terminalStatusIDs := map[models.PipelineStatus]string{
		models.StatusCompleted: "20202020-2020-2020-2020-202020202020",
		models.StatusCancelled: "30303030-3030-3030-3030-303030303030",
		models.StatusFailed:    "40404040-4040-4040-4040-404040404040",
	}
	for status, id := range terminalStatusIDs {
		status, id := status, id
		t.Run("terminal status "+string(status), func(t *testing.T) {
			pipeline := models.Pipeline{ID: id, Status: status}
			db.Create(&pipeline)

			w := doRequest(r, http.MethodPatch, "/pipelines/"+pipeline.ID+"/cancel", nil)
			if w.Code != http.StatusConflict {
				t.Errorf("status = %d, want %d", w.Code, http.StatusConflict)
			}
		})
	}

	t.Run("pending pipeline is cancelled immediately", func(t *testing.T) {
		pipeline := models.Pipeline{ID: "50505050-5050-5050-5050-505050505050", Status: models.StatusPending}
		db.Create(&pipeline)

		w := doRequest(r, http.MethodPatch, "/pipelines/"+pipeline.ID+"/cancel", nil)
		if w.Code != http.StatusOK {
			t.Fatalf("status = %d, want %d", w.Code, http.StatusOK)
		}

		var stored models.Pipeline
		db.First(&stored, "id = ?", pipeline.ID)
		if stored.Status != models.StatusCancelled {
			t.Errorf("status = %q, want %q", stored.Status, models.StatusCancelled)
		}
	})

	t.Run("processing pipeline notifies worker and leaves status untouched", func(t *testing.T) {
		pipeline := models.Pipeline{ID: "60606060-6060-6060-6060-606060606060", Status: models.StatusProcessing}
		db.Create(&pipeline)

		w := doRequest(r, http.MethodPatch, "/pipelines/"+pipeline.ID+"/cancel", nil)
		if w.Code != http.StatusOK {
			t.Fatalf("status = %d, want %d", w.Code, http.StatusOK)
		}

		var stored models.Pipeline
		db.First(&stored, "id = ?", pipeline.ID)
		if stored.Status != models.StatusProcessing {
			t.Errorf("status = %q, want %q (worker owns the transition)", stored.Status, models.StatusProcessing)
		}

		if !fw.wasCancelled(pipeline.ID) {
			t.Errorf("expected worker cancel endpoint to be called for pipeline %q", pipeline.ID)
		}
	})
}

func TestDeletePipeline(t *testing.T) {
	db := setupTestDB(t)
	r := newTestRouter()

	t.Run("not found", func(t *testing.T) {
		w := doRequest(r, http.MethodDelete, "/pipelines/12121212-1212-1212-1212-121212121212", nil)
		if w.Code != http.StatusNotFound {
			t.Errorf("status = %d, want %d", w.Code, http.StatusNotFound)
		}
	})

	t.Run("deletes existing pipeline", func(t *testing.T) {
		pipeline := models.Pipeline{ID: "13131313-1313-1313-1313-131313131313", Status: models.StatusPending}
		db.Create(&pipeline)

		w := doRequest(r, http.MethodDelete, "/pipelines/"+pipeline.ID, nil)
		if w.Code != http.StatusNoContent {
			t.Errorf("status = %d, want %d", w.Code, http.StatusNoContent)
		}

		var count int64
		db.Model(&models.Pipeline{}).Where("id = ?", pipeline.ID).Count(&count)
		if count != 0 {
			t.Error("expected pipeline to be removed from the database")
		}
	})
}
