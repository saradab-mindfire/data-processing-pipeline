package models

import (
	"time"

	"github.com/google/uuid"
	"gorm.io/gorm"
)

type PipelineStatus string

const (
	StatusPending    PipelineStatus = "pending"
	StatusProcessing PipelineStatus = "processing"
	StatusCompleted  PipelineStatus = "completed"
	StatusFailed     PipelineStatus = "failed"
	StatusCancelled  PipelineStatus = "cancelled"
)

type Pipeline struct {
	ID               string         `json:"id" gorm:"primaryKey"`
	Name             string         `json:"name"`
	Status           PipelineStatus `json:"status"`
	Description      string         `json:"description"`
	TotalRecords     int            `json:"total_records"`
	ProcessedRecords int            `json:"processed_records"`
	ValidRecords     int            `json:"valid_records"`
	InvalidRecords   int            `json:"invalid_records"`
	StartedAt        time.Time      `json:"started_at"`
	CompletedAt      time.Time      `json:"completed_at"`
}

func (p *Pipeline) BeforeCreate(tx *gorm.DB) error {
	if p.ID == "" {
		p.ID = uuid.NewString()
	}
	if p.Status == "" {
		p.Status = StatusPending
	}
	return nil
}