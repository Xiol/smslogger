package main

import (
	"fmt"
	"os"
	"time"

	log "github.com/Sirupsen/logrus"
	_ "github.com/go-sql-driver/mysql"
	"github.com/jinzhu/gorm"
)

type Sms struct {
	ID        int
	Timestamp time.Time
	From      string
	Message   string
	Hash      string `sql:"unique_index"`
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

	db, err := gorm.Open("mysql", fmt.Sprintf("%s:%s@tcp(%s:3306)/%s?charset=utf8&parseTime=True&loc=Local", user, pass, host, database))
	if err != nil {
		log.Fatalf("Error connecting to database: %s", err)
	}

	db.Set("gorm:table_options", "ENGINE=InnoDB").AutoMigrate(&Sms{})
	db.DB().SetMaxIdleConns(0)
	db.DB().SetMaxOpenConns(50)

	return &db
}
