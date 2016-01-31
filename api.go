package main

import (
	"crypto/sha256"
	"fmt"
	"time"

	log "github.com/Sirupsen/logrus"
	"github.com/gin-gonic/gin"
	"github.com/jinzhu/gorm"
)

type Api struct {
	db *gorm.DB
}

func InitApi(engine *gin.Engine, db *gorm.DB) *Api {
	a := &Api{}
	a.db = db

	engine.POST("/sms", a.AddSMS)
	engine.GET("/sms", a.GetSMS)
	engine.GET("/search", a.SearchSMS)

	return a
}

func (a *Api) AddSMS(c *gin.Context) {
	log.Debug("Adding entry...")

	message := c.PostForm("message")
	mtime := c.PostForm("time")
	mdate := c.PostForm("date")
	from := c.PostForm("from")

	if message == "" {
		stringWebError(c, 400, "Message data missing")
		return
	}

	log.Infof("SMS Received: Time: %s, Date: %s, From: %s, Message: %s", mtime, mdate, from, message)

	timestamp, err := time.Parse("02-01-2006 15.04", fmt.Sprintf("%s %s", mdate, mtime))
	if err != nil {
		stringWebError(c, 400, "Failed to parse time")
		return
	}

	hasher := sha256.New()
	hasher.Write([]byte(timestamp.Format("2006-01-02 15:04:05")))
	hasher.Write([]byte(from))
	hasher.Write([]byte(message))

	sms := &Sms{
		Timestamp: timestamp,
		From:      from,
		Message:   message,
		Hash:      fmt.Sprintf("%x", hasher.Sum(nil)),
	}

	a.db.Save(sms)

	if a.db.NewRecord(sms) {
		c.String(500, "Error: Failed to store")
		return
	}

	log.Debugf("Saved to database")
	c.String(201, "Stored")
    return
}

func (a *Api) GetSMS(c *gin.Context) {
	log.Debug("Getting entry...")

	start := c.Query("start")
	limit := c.Query("limit")
	id := c.Query("id")

	if id != "" {
		sms := Sms{}
		a.db.Find(&sms, "id = ?", id)
		log.Infof("Retrieved SMS with ID %s", id)
		c.JSON(200, sms)
		return
	}

	if start != "" {
		if limit == "" {
			limit = "50"
		}

		smss := make([]*Sms, 0)
		a.db.Offset(start).Limit(limit).Find(&smss)

		log.Infof("Retrieved %d results, returning up to %s to client from offset %s", len(smss), limit, start)

		c.JSON(200, smss)
		return
	}

	jsonWebError(c, 400, "Bad request, missing 'id' or 'start' query")
	return
}

func (a *Api) SearchSMS(c *gin.Context) {
	log.Debug("Searching...")

	query := c.Query("q")
	start := c.Query("start")
	limit := c.Query("limit")

	if query == "" {
		jsonWebError(c, 400, "Bad request, missing query")
		return
	}

	if start == "" {
		start = "0"
	}

	if limit == "" {
		limit = "50"
	}

	smss := make([]*Sms, 0)
	a.db.Offset(start).Limit(limit).Where("message LIKE ?", fmt.Sprintf("%%%s%%", query)).Find(&smss)

	log.Infof("Search for '%s' returned %d results, returning up to %s to client from offset %s", query, len(smss), limit, start)

	c.JSON(200, smss)
	return
}
