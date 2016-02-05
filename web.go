package main

import (
	"fmt"
	"time"

	log "github.com/Sirupsen/logrus"
	"github.com/gin-gonic/gin"
)

func InitWeb(logLevel, assets string) *gin.Engine {
	if logLevel != "debug" {
		gin.SetMode(gin.ReleaseMode)
	}

	engine := gin.New()
	engine.HandleMethodNotAllowed = true
	engine.RedirectTrailingSlash = true
	engine.Use(gin.Recovery())
	engine.Static("/static", assets)

	engine.Use(func(c *gin.Context) {
		start := time.Now()

		c.Next()

		end := time.Now()
		latency := end.Sub(start)
		clientIP := c.Request.RemoteAddr
		method := c.Request.Method
		statusCode := c.Writer.Status()
		errors := ""
		if c.Errors != nil {
			errors = fmt.Sprintf(" - %s", c.Errors.String())
		}

		log.Infof("[GIN] %v - %3d - %12v - %s %s %s%s",
			end.Format("2006-01-02 15:04:05"),
			statusCode,
			latency,
			clientIP,
			method,
			c.Request.URL.Path,
			errors,
		)
	})

	engine.Use(func(c *gin.Context) {
		c.Writer.Header().Set("Server", "SMS Logger")
		c.Next()
	})

	return engine
}

func jsonWebError(c *gin.Context, status int, message string, args ...interface{}) {
	log.Errorf("Error (%d): %s", status, fmt.Sprintf(message, args...))
	c.JSON(status, gin.H{"output": fmt.Sprintf(message, args...)})
	c.Abort()
}

func stringWebError(c *gin.Context, status int, message string, args ...interface{}) {
	log.Errorf("Error (%d): %s", status, fmt.Sprintf(message, args...))
	c.String(status, message, args)
	c.Abort()
}
