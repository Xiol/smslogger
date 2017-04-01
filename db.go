package main

import (
	"fmt"
	"html/template"
	"os"
	"time"

	log "github.com/Sirupsen/logrus"
	"github.com/jinzhu/gorm"
	_ "github.com/lib/pq"
)

type Sms struct {
	ID          int
	Timestamp   time.Time
	From        string
	Message     string        `sql:"type:text"`
	MessageHTML template.HTML `sql:"-" json:"-"`
	Hash        string        `sql:"type:char(64);unique_index"`
}

func InitDB() *gorm.DB {
	log.Debug("Initialising database...")

	database := os.Getenv("SMS_DATABASE")
	user := os.Getenv("SMS_DBUSER")
	pass := os.Getenv("SMS_DBPASS")
	host := os.Getenv("SMS_DBHOST")

	if database == "" || user == "" || pass == "" {
		log.Fatal("Missing database environment variable")
	}

	if host == "" {
		host = "127.0.0.1"
	}

	db, err := gorm.Open("postgres", fmt.Sprintf("user=%s password=%s dbname=%s host=%s sslmode=disable", user, pass, database, host))
	if err != nil {
		log.Fatalf("Error connecting to database: %s", err)
	}

	db.AutoMigrate(&Sms{})

	return db
}
