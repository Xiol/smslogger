package main

import (
	"crypto/sha256"
	"fmt"
	"html/template"
	"strconv"
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

	engine.LoadHTMLGlob("templates/*")

	engine.GET("/", a.WUI)
	engine.POST("/sms", a.AddSMS)
	engine.GET("/sms", a.GetSMS)
	engine.GET("/search", a.SearchSMS)

	return a
}

func (a *Api) WUI(c *gin.Context) {
	smss, err := a.getSms(c)
	if err != nil {
		// do what here?
		c.AbortWithStatus(400)
	}

	var start, limit int64
	var sstart, slimit, q string
	q = c.Query("q")

	smscount := a.countSms(q)

	sstart = c.Query("start")
	slimit = c.Query("limit")

	if sstart == "" {
		start = 0
	} else {
		start, err = strconv.ParseInt(sstart, 10, 64)
		if err != nil {
			stringWebError(c, 400, "start must be an integer")
			return
		}
	}

	if slimit == "" {
		limit = 50
	} else {
		limit, err = strconv.ParseInt(slimit, 10, 64)
		if err != nil {
			stringWebError(c, 400, "limit must be an integer")
			return
		}
	}

	morePrev := start > 0
	moreNext := (start + limit) < int64(smscount)

	if q != "" {
		q = fmt.Sprintf("&q=%s", template.URLQueryEscaper(q))
	}

	log.Debugf("start: %v, limit %v, moreNext: %v, morePrev: %v, smsCount: %d", start, limit, moreNext, morePrev, smscount)

	c.HTML(200, "index.tmpl", gin.H{
		"SMS":       smss,
		"Count":     smscount,
		"MorePrev":  morePrev,
		"MoreNext":  moreNext,
		"Limit":     limit,
		"StartNext": start + limit,
		"StartPrev": start - limit,
		"Query":     template.URL(q),
	})
	return
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
	log.Debug("Getting SMS's...")

	smss, err := a.getSms(c)
	if err != nil {
		jsonWebError(c, 400, err.Error())
		return
	}

	c.JSON(200, smss)
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
	a.db.Offset(start).Limit(limit).Where("message LIKE ?", fmt.Sprintf("%% %s %%", query)).Find(&smss)

	log.Infof("Search for '%s' returned %d results, returning up to %s to client from offset %s", query, len(smss), limit, start)

	c.JSON(200, smss)
	return
}

func (a *Api) countSms(q string) int {
	var count int
	if q == "" {
		a.db.Table("sms").Count(&count)
	} else {
		a.db.Table("sms").Where("message LIKE ?", fmt.Sprintf("%% %s %%", q)).Count(&count)
	}
	return count
}

func (a *Api) getSms(c *gin.Context) ([]*Sms, error) {
	start := c.Query("start")
	limit := c.Query("limit")
	id := c.Query("id")
	q := c.Query("q")

	var smss []*Sms

	if id != "" {
		sms := Sms{}
		a.db.Find(&sms, "id = ?", id)
		log.Infof("Retrieved SMS with ID %s", id)
		smss = make([]*Sms, 1)
		smss[0] = &sms
		return smss, nil
	}

	if start == "" {
		start = "0"
	}

	if limit == "" {
		limit = "50"
	}

	if q != "" {
		a.db.Offset(start).Limit(limit).Order("timestamp desc").Where("message LIKE ?", fmt.Sprintf("%% %s %%", q)).Find(&smss)
	} else {
		a.db.Offset(start).Limit(limit).Order("timestamp desc").Find(&smss)
	}

	log.Infof("Retrieved %d results, returning up to %s to client from offset %s", len(smss), limit, start)

	return smss, nil
}
