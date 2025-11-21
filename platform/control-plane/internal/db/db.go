package db

import (
	"log"

	"gorm.io/driver/postgres"
	"gorm.io/gorm"
)

func Init(databaseURL string) (*gorm.DB, error) {
	db, err := gorm.Open(postgres.Open(databaseURL), &gorm.Config{})
	if err != nil {
		return nil, err
	}

	log.Println("Connected to database")
	return db, nil
}
