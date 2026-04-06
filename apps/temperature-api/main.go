package main

import (
	"context"
	"log"
	"math/rand"
	"net/http"
	"os"
	"os/signal"
	"syscall"
	"time"

	"github.com/gin-gonic/gin"
)

func main() {
	service_name := "temperature-api"

	log.Printf("%s starting\n", service_name)

	// Initialize router
	router := gin.Default()

	// Health check endpoint
	router.GET("/health", func(c *gin.Context) {
		c.JSON(http.StatusOK, gin.H{
			"status": "ok",
		})
	})

	// Get temperature endpoint
	router.GET("/temperature", func(c *gin.Context) {
		location := c.Query("location")
		sensorID := c.Query("sensor_id")

		// If no location is provided, use a default based on sensor ID
		if location == "" {
			switch sensorID {
			case "1":
				location = "Living Room"
			case "2":
				location = "Bedroom"
			case "3":
				location = "Kitchen"
			default:
				location = "Unknown"
			}
		}

		// If no sensor ID is provided, generate one based on location
		if sensorID == "" {
			switch location {
			case "Living Room":
				sensorID = "1"
			case "Bedroom":
				sensorID = "2"
			case "Kitchen":
				sensorID = "3"
			default:
				sensorID = "0"
			}
		}

		value := 15.0 + rand.Float64()*15.0 // Random value between 15 and 30

		reply := map[string]any{
			"status":      "ok",
			"value":       value,
			"unit":        "C",
			"location":    location,
			"timestamp":   time.Now(),
			"sensor_type": "temperature_C",
			"sensor_id":   sensorID,
			"description": "Random temperature from API",
		}
		c.JSON(http.StatusOK, reply)
	})

	// Sensor ID specific endpoint
	router.GET("/temperature/:sensor_id", func(c *gin.Context) {
		sensorID := c.Param("sensor_id")
		location := "Unknown"
		switch sensorID {
		case "1":
			location = "Living Room"
		case "2":
			location = "Bedroom"
		case "3":
			location = "Kitchen"
		}

		value := 15.0 + rand.Float64()*15.0

		reply := map[string]any{
			"status":      "ok",
			"value":       value,
			"unit":        "C",
			"location":    location,
			"timestamp":   time.Now(),
			"sensor_type": "temperature_C",
			"sensor_id":   sensorID,
			"description": "Random temperature from API",
		}
		c.JSON(http.StatusOK, reply)
	})

	// Start server
	srv := &http.Server{
		Addr:    "0.0.0.0:" + getEnv("PORT", "8081"),
		Handler: router,
	}

	go func() {
		log.Printf("Service %s starting on %s\n", service_name, srv.Addr)
		if err := srv.ListenAndServe(); err != nil && err != http.ErrServerClosed {
			log.Fatalf("Failed to start server: %v\n", err)
		}
	}()

	quit := make(chan os.Signal, 1)
	signal.Notify(quit, syscall.SIGINT, syscall.SIGTERM)
	<-quit
	log.Println("Shutting down server...")

	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()
	if err := srv.Shutdown(ctx); err != nil {
		log.Fatalf("Service forced to shutdown: %v\n", err)
	}

	log.Printf("%s exited properly\n", service_name)
}

func getEnv(key, defaultValue string) string {
	value := os.Getenv(key)
	if value == "" {
		return defaultValue
	}
	return value
}
