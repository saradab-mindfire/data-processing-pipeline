package worker

import (
	"context"
	"fmt"
	"os"
	"path/filepath"
	"testing"

	"github.com/glebarez/sqlite"
	"github.com/google/uuid"
	"gorm.io/gorm"

	"github.com/saradab-mindfire/data-processing-pipeline/apps/worker/dataio"
	"github.com/saradab-mindfire/data-processing-pipeline/packages/database"
	"github.com/saradab-mindfire/data-processing-pipeline/packages/models"
)

func setupRunTestDB(t *testing.T) (db *gorm.DB, exportDir string) {
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

	exportDir = t.TempDir()
	dataio.InitExportsDir(exportDir)
	t.Cleanup(func() { dataio.InitExportsDir("exports") })

	return db, exportDir
}

func TestRunClaimsPendingPipelineAndCompletes(t *testing.T) {
	db, _ := setupRunTestDB(t)

	pipeline := models.Pipeline{ID: "run-pending-pipeline", Status: models.StatusPending}
	db.Create(&pipeline)

	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()
	j := &job{pipelineID: pipeline.ID, ctx: ctx, cancel: cancel}
	jobsMu.Lock()
	jobs[pipeline.ID] = j
	jobsMu.Unlock()

	run(j, pipeline.ID, models.PipelineRequest{})

	var stored models.Pipeline
	if err := db.First(&stored, "id = ?", pipeline.ID).Error; err != nil {
		t.Fatalf("failed to load pipeline: %v", err)
	}
	if stored.Status != models.StatusCompleted {
		t.Errorf("status = %q, want %q", stored.Status, models.StatusCompleted)
	}

	jobsMu.Lock()
	_, stillTracked := jobs[pipeline.ID]
	jobsMu.Unlock()
	if stillTracked {
		t.Error("expected run to remove the job once finished")
	}
}

func TestRunSkipsPipelineNotInPendingStatus(t *testing.T) {
	db, _ := setupRunTestDB(t)

	pipeline := models.Pipeline{ID: "run-already-cancelled-pipeline", Status: models.StatusCancelled}
	db.Create(&pipeline)

	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()
	j := &job{pipelineID: pipeline.ID, ctx: ctx, cancel: cancel}
	jobsMu.Lock()
	jobs[pipeline.ID] = j
	jobsMu.Unlock()

	run(j, pipeline.ID, models.PipelineRequest{})

	var stored models.Pipeline
	if err := db.First(&stored, "id = ?", pipeline.ID).Error; err != nil {
		t.Fatalf("failed to load pipeline: %v", err)
	}
	if stored.Status != models.StatusCancelled {
		t.Errorf("status = %q, want %q (run must not process a non-pending pipeline)", stored.Status, models.StatusCancelled)
	}

	jobsMu.Lock()
	_, stillTracked := jobs[pipeline.ID]
	jobsMu.Unlock()
	if stillTracked {
		t.Error("expected run to remove the job even when it declines to process")
	}
}

func TestRunMarksCancelledPipelineWithoutExporting(t *testing.T) {
	db, exportDir := setupRunTestDB(t)

	pipeline := models.Pipeline{ID: "run-cancel-during-processing-pipeline", Status: models.StatusPending}
	db.Create(&pipeline)

	ctx, cancel := context.WithCancel(context.Background())
	cancel() // simulate the pipeline being cancelled before processing finishes
	j := &job{pipelineID: pipeline.ID, ctx: ctx, cancel: cancel}
	jobsMu.Lock()
	jobs[pipeline.ID] = j
	jobsMu.Unlock()

	run(j, pipeline.ID, models.PipelineRequest{})

	var stored models.Pipeline
	if err := db.First(&stored, "id = ?", pipeline.ID).Error; err != nil {
		t.Fatalf("failed to load pipeline: %v", err)
	}
	if stored.Status != models.StatusCancelled {
		t.Errorf("status = %q, want %q", stored.Status, models.StatusCancelled)
	}

	if _, err := os.Stat(filepath.Join(exportDir, pipeline.ID+".json")); !os.IsNotExist(err) {
		t.Error("expected no export file to be written for a cancelled pipeline")
	}
}
