package main

import (
	"bytes"
	"crypto/sha256"
	"encoding/csv"
	"fmt"
	"html/template"
	"io"
	"strconv"
	"strings"
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
	engine.GET("/export", a.ExportUI)
	engine.GET("/export/do", a.DoExport)

	return a
}

func (a *Api) WUI(c *gin.Context) {
	smss, errcode := a.getSms(c)
	if errcode > 0 {
		if errcode == 404 {
			c.HTML(404, "index.tmpl", gin.H{
				"NotFound": true,
			})
			return
		} else {
			c.AbortWithStatus(errcode)
		}
	}

	var err error
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

func (a *Api) ExportUI(c *gin.Context) {
	c.HTML(200, "export.tmpl", gin.H{"ExportPage": true})
	return
}

func (a *Api) DoExport(c *gin.Context) {
	sto := c.Query("to")
	sfrom := c.Query("from")
	q := c.Query("q")

	log.Debugf("Export request: From: %s, To: %s, Query: %s", sfrom, sto, q)

	var err1, err2 error
	var to, from time.Time
	if sto == "" {
		to = time.Time{}
	} else {
		to, err1 = time.Parse("2006-01-02", sto)
	}

	if sfrom != "" {
		from, err2 = time.Parse("2006-01-02", sfrom)
	} else {
		from = time.Time{}
	}

	if err1 != nil || err2 != nil {
		stringWebError(c, 400, "Incorrect date format, must be YYYY-MM-DD. %s %s", err1, err2)
		return
	}

	buf := bytes.Buffer{}
	a.export(&buf, to, from, q)

	var qs, tos, froms string
	if q != "" {
		qs = fmt.Sprintf("_%s", strings.Replace(q, " ", "_", -1))
	}
	if to.IsZero() {
		tos = time.Now().Format("2006-01-02")
	} else {
		tos = sto
	}
	if !from.IsZero() {
		froms = fmt.Sprintf("%s_", sfrom)
	}

	c.Header("Content-Disposition", fmt.Sprintf("inline; filename=\"sms-%s%s%s.csv\"", froms, tos, qs))
	c.Data(200, "text/csv; charset=utf-8", buf.Bytes())
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

	tx := a.db.Begin()

	if err := tx.Create(sms).Error; err != nil {
		tx.Rollback()
		stringWebError(c, 500, "Error creating entry: %s", err)
		return
	}

	tx.Commit()

	log.Debugf("Saved to database")
	c.String(201, "Stored")
	return
}

func (a *Api) GetSMS(c *gin.Context) {
	log.Debug("Getting SMS's...")

	smss, errcode := a.getSms(c)
	if errcode > 0 {
		if errcode == 404 {
			jsonWebError(c, errcode, "Record not found.")
		} else {
			jsonWebError(c, errcode, "Error, please see logs.")
		}
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

func (a *Api) getSms(c *gin.Context) ([]*Sms, int) {
	start := c.Query("start")
	limit := c.Query("limit")
	id := c.Query("id")
	q := c.Query("q")
	hash := c.Query("hash")

	var smss []*Sms

	if id != "" || hash != "" {
		sms := Sms{}

		if id != "" {
			if a.db.Find(&sms, "id = ?", id).RecordNotFound() {
				return nil, 404
			}
		}

		if hash != "" {
			if a.db.Find(&sms, "hash = ?", hash).RecordNotFound() {
				return nil, 404
			}
		}

		log.Infof("Retrieved SMS with ID %s", id)
		smss = make([]*Sms, 1)
		smss[0] = &sms
		return smss, 0
	}

	if start == "" {
		start = "0"
	}

	if limit == "" {
		limit = "50"
	}

	if q != "" {
		if a.db.Offset(start).Limit(limit).Order("timestamp desc").Where("message LIKE ?", fmt.Sprintf("%% %s %%", q)).Find(&smss).RecordNotFound() {
			return nil, 404
		}
	} else {
		if a.db.Offset(start).Limit(limit).Order("timestamp desc").Find(&smss).RecordNotFound() {
			return nil, 404
		}
	}

	if len(smss) == 0 {
		return nil, 404
	}

	log.Infof("Retrieved %d results, returning up to %s to client from offset %s", len(smss), limit, start)

	return smss, 0
}

func (a *Api) export(out io.Writer, from, to time.Time, query string) error {
	w := csv.NewWriter(out)

	if to.IsZero() && !from.IsZero() {
		to = time.Now()
	}

	var smss []*Sms

	if query != "" {
		if !from.IsZero() {
			a.db.Where("timestamp < ? AND timestamp > ? AND message LIKE ?", from, to, fmt.Sprintf("%% %s %%", query)).Find(&smss)
		} else {
			a.db.Where("message LIKE ?", fmt.Sprintf("%% %s %%", query)).Find(&smss)
		}
	} else {
		if !from.IsZero() {
			a.db.Where("timestamp < ? AND timestamp > ?", from, to).Find(&smss)
		} else {
			a.db.Find(&smss)
		}
	}

	record := make([]string, 3)
	for i := range smss {
		record[0] = smss[i].Timestamp.Format("2006-01-02 15:04")
		record[1] = smss[i].From
		record[2] = smss[i].Message
		err := w.Write(record)
		if err != nil {
			return err
		}
	}
	w.Flush()
	return nil
}
