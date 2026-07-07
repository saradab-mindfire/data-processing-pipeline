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
	ID							string	 		`json:"id" gorm:"primaryKey"`
	NAME						string	 		`json:"name"`
	STATUS						PipelineStatus	`json:"status"`
	DESCRIPTION					string	 		`json:"description"`
	TOTAL_RECORDS				int    	 	`json:"total_records"`
	PROCESSED_RECORDS			int    	 	`json:"processed_records"`
	VALID_RECORDS				int    	 	`json:"valid_records"`
	INVALID_RECORDS				int    	 	`json:"invalid_records"`
	STARTED_AT					time.Time	`json:"started_at"`
	COMPLETED_AT				time.Time	`json:"completed_at"`
}

func (p *Pipeline) BeforeCreate(tx *gorm.DB) error {
	if p.ID == "" {
		p.ID = uuid.NewString()
	}
	if p.STATUS == "" {
		p.STATUS = StatusPending
	}
	return nil
}