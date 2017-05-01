package main

import (
	"fmt"
	"html/template"
	"os"
	"time"

	log "github.com/Sirupsen/logrus"
	_ "github.com/go-sql-driver/mysql"
	"github.com/jinzhu/gorm"
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

	db, err := gorm.Open("mysql", fmt.Sprintf("%s:%s@tcp(%s:3306)/%s?parseTime=True&loc=Local", user, pass, host, database))
	if err != nil {
		log.Fatalf("Error connecting to database: %s", err)
	}

	db.AutoMigrate(&Sms{})

	return db
}
