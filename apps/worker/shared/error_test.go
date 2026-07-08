package shared

import (
	"fmt"
	"testing"

	"github.com/glebarez/sqlite"
	"github.com/google/uuid"
	"gorm.io/gorm"

	"github.com/saradab-mindfire/data-processing-pipeline/packages/database"
	"github.com/saradab-mindfire/data-processing-pipeline/packages/models"
)

func TestSaveErrorNoopWhenNoDatabase(t *testing.T) {
	previous := database.Instance
	database.Instance = nil
	t.Cleanup(func() { database.Instance = previous })

	// Must not panic when no database connection is configured.
	SaveError("some-pipeline", "boom")
}

func TestSaveErrorPersistsRecord(t *testing.T) {
	dsn := fmt.Sprintf("file:%s?mode=memory&cache=shared", uuid.NewString())
	db, err := gorm.Open(sqlite.Open(dsn), &gorm.Config{})
	if err != nil {
		t.Fatalf("failed to open in-memory database: %v", err)
	}
	if err := db.AutoMigrate(&models.PipelineError{}); err != nil {
		t.Fatalf("failed to migrate in-memory database: %v", err)
	}

	previous := database.Instance
	database.Instance = db
	t.Cleanup(func() { database.Instance = previous })

	SaveError("pipeline-123", "something went wrong")

	var errs []models.PipelineError
	if err := db.Where("pipeline_id = ?", "pipeline-123").Find(&errs).Error; err != nil {
		t.Fatalf("failed to query pipeline errors: %v", err)
	}
	if len(errs) != 1 {
		t.Fatalf("got %d errors, want 1", len(errs))
	}
	if errs[0].Message != "something went wrong" {
		t.Errorf("message = %q, want %q", errs[0].Message, "something went wrong")
	}
	if errs[0].ID == "" {
		t.Error("expected BeforeCreate to populate an ID")
	}
}
