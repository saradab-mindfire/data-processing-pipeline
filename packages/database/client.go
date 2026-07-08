package database

import (
	"log"
	"time"

	"github.com/saradab-mindfire/data-processing-pipeline/packages/models"
	"gorm.io/driver/postgres"
	"gorm.io/gorm"
)

const (
	maxOpenConns    = 25
	maxIdleConns    = 10
	connMaxLifetime = 30 * time.Minute
	connMaxIdleTime = 5 * time.Minute
)

var Instance *gorm.DB
var dbError error

func Connect(connectionString string) {
	Instance, dbError = gorm.Open(postgres.Open(connectionString), &gorm.Config{})
	if dbError != nil {
		log.Fatal(dbError)
		panic("Cannot connect to DB")
	}

	sqlDB, err := Instance.DB()
	if err != nil {
		log.Fatal(err)
		panic("Cannot access underlying DB connection pool")
	}
	sqlDB.SetMaxOpenConns(maxOpenConns)
	sqlDB.SetMaxIdleConns(maxIdleConns)
	sqlDB.SetConnMaxLifetime(connMaxLifetime)
	sqlDB.SetConnMaxIdleTime(connMaxIdleTime)

	log.Println("Connected to Database!")
}

func Migrate() {
	Instance.AutoMigrate(&models.Pipeline{}, &models.PipelineError{})
	log.Println("Database Migration Completed!")
}