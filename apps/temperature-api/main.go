package main

import (
	"context"
	"log"
	"math"
	"net/http"
	"os"
	"os/signal"
	"syscall"
	"time"

	"github.com/gin-gonic/gin"
)

// TemperatureResponse represents the response from the temperature API
type TemperatureResponse struct {
	Value       float64   `json:"value"`
	Unit        string    `json:"unit"`
	Timestamp   time.Time `json:"timestamp"`
	Location    string    `json:"location"`
	Status      string    `json:"status"`
	SensorID    string    `json:"sensor_id"`
	SensorType  string    `json:"sensor_type"`
	Description string    `json:"description"`
}

func main() {
	// Set up database connection
	//dbURL := getEnv("DATABASE_URL", "postgres://postgres:postgres@localhost:5432/smarthome")

	service_name := "temperatire-api"

	log.Printf("%s starting\n", service_name)

	// Initialize router
	router := gin.Default()

	// Health check endpoint
	router.GET("/health", func(c *gin.Context) {
		c.JSON(http.StatusOK, gin.H{
			"status": "ok",
		})
	})

	t := 0.0

	// Health check endpoint
	router.GET("/temperature", func(c *gin.Context) {
		t = math.Mod(t+0.01, 100)
		reply := map[string]any{}

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

		reply["status"] = "ok"
		reply["value"] = t
		reply["unit"] = "C"
		reply["location"] = location
		reply["timestamp"] = time.Now()
		reply["sensor_type"] = "temperature_C"
		reply["sensor_id"] = sensorID
		reply["description"] = "I hope you notice that line and send me hello"
		c.JSON(http.StatusOK, reply)
	})

	// API routes
	//apiRoutes := router.Group("/api/v1")

	// Start server
	srv := &http.Server{
		Addr:    getEnv("PORT", ":8081"),
		Handler: router,
	}

	// Start the server in a goroutine
	go func() {
		log.Printf("Service %s starting on %s\n", service_name, srv.Addr)
		if err := srv.ListenAndServe(); err != nil && err != http.ErrServerClosed {
			log.Fatalf("Failed to start server: %v\n", err)
		}
	}()

	// Wait for interrupt signal to gracefully shut down the server
	quit := make(chan os.Signal, 1)
	signal.Notify(quit, syscall.SIGINT, syscall.SIGTERM)
	<-quit
	log.Println("Shutting down server...")

	// Create a deadline for server shutdown
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()
	if err := srv.Shutdown(ctx); err != nil {
		log.Fatalf("Service forced to shutdown: %v\n", err)
	}

	log.Printf("%s exited properly\n", service_name)
}

// getEnv gets an environment variable or returns a default value
func getEnv(key, defaultValue string) string {
	value := os.Getenv(key)
	if value == "" {
		return defaultValue
	}
	return value
}
