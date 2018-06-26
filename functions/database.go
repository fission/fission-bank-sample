package main

import (
	"fmt"
	"log"

	"github.com/jinzhu/gorm"
)

var dbConn *gorm.DB

func getDB() *gorm.DB {
	if dbConn == nil {
		dbUrl := "postgresql://root@cockroachdb.bank:26257/bank?sslmode=disable" //os.Getenv("DB_URL")
		dbName := "bank"                                                         //os.Getenv("DB_NAME")
		dbConn = connectDB(dbUrl, dbName)
	}
	return dbConn.Debug()
}

// connectDB establishes connection to database
func connectDB(dbUrl, dbName string) *gorm.DB {
	conn, err := gorm.Open("postgres", dbUrl)
	if err != nil {
		log.Fatalf("Error establishing connection to database(%v): %v", dbUrl, err)
	}
	initDB(conn, dbName)
	return conn
}

func initDB(db *gorm.DB, dbName string) {
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
	for _, model := range []interface{}{&Account{}, &TransactionRecord{}, &AccountBalance{}, &Session{}} {
		err = db.AutoMigrate(model).Error
		if err != nil {
			log.Fatalf("Error initializing database: %v", err)
		}
	}
}
