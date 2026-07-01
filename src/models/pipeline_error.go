package models

import (
	"time"

	"github.com/google/uuid"
	"gorm.io/gorm"
)

type PipelineError struct {
	ID         string    `json:"id" gorm:"primaryKey"`
	PipelineID string    `json:"pipeline_id" gorm:"index;not null"`
	RecordID   string    `json:"record_id"`
	Message    string    `json:"message"`
	CreatedAt  time.Time `json:"created_at"`
}

func (e *PipelineError) BeforeCreate(tx *gorm.DB) error {
	if e.ID == "" {
		e.ID = uuid.NewString()
	}
	return nil
}
