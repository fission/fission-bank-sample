package main

import (
	"fmt"
	"log"

	"github.com/jinzhu/gorm"
	_ "github.com/jinzhu/gorm/dialects/postgres"
)

// ConnectDB establishes connection to database
func ConnectDB(dbUrl, dbName string) *gorm.DB {
	conn, err := gorm.Open("postgres", dbUrl)
	if err != nil {
		log.Fatalf("Error establishing connection to database(%v): %v", dbUrl, err)
	}
	InitDB(conn, dbName)
	return conn
}

func InitDB(db *gorm.DB, dbName string) {
	if db == nil {
		log.Fatal("Error initializing database with nil db connection")
	}

	// Create database
	query := fmt.Sprintf("CREATE DATABASE IF NOT EXISTS %s", dbName)
	_, err := db.DB().Exec(query)
	if err != nil {
		log.Fatalf("Error creating database: %v", err)
	}

	// Create tables based on models
	for _, model := range []interface{}{&Message{}} {
		err = db.AutoMigrate(model).Error
		if err != nil {
			log.Fatalf("Error initializing database: %v", err)
		}
	}
}
