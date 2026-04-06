package main

import (
	"database/sql"
	"encoding/json"
	"log"
	"os"
	"os/signal"
	"syscall"
	"time"

	mqtt "github.com/eclipse/paho.mqtt.golang"
	_ "github.com/lib/pq"
)

// SensorData represents the telemetry payload
type SensorData struct {
	SensorID  string  `json:"sensor_id"`
	Temp      float64 `json:"temperature"`
	Humidity  float64 `json:"humidity"`
	Status    string  `json:"status"`
	Timestamp int64   `json:"timestamp"`
}

var db *sql.DB

func main() {
	// 1. Database Configuration
	dbURL := getEnv("DATABASE_URL", "postgres://postgres:postgres@postgres:5432/smarthome?sslmode=disable")
	var err error
	for i := 0; i < 10; i++ {
		db, err = sql.Open("postgres", dbURL)
		if err == nil {
			err = db.Ping()
		}
		if err == nil {
			break
		}
		log.Printf("Waiting for DB... %v\n", err)
		time.Sleep(2 * time.Second)
	}
	if err != nil {
		log.Fatalf("Unable to connect to database: %v\n", err)
	}
	defer db.Close()
	log.Println("Connected to database successfully")

	// 2. Broker Configuration
	mqttURL := getEnv("MQTT_URL", "tcp://mosquitto:1883")
	opts := mqtt.NewClientOptions().AddBroker(mqttURL)
	opts.SetClientID("telemetry_service")
	opts.SetCleanSession(true)

	// Callback for when a message is received
	opts.SetDefaultPublishHandler(func(client mqtt.Client, msg mqtt.Message) {
		log.Printf("Received message on topic: %s\n", msg.Topic())
		var data SensorData
		if err := json.Unmarshal(msg.Payload(), &data); err != nil {
			log.Printf("Error unmarshaling payload: %v\n", err)
			return
		}

		// Save to DB
		query := `INSERT INTO telemetry (sensor_id, temperature, humidity, status, timestamp) 
		          VALUES ($1, $2, $3, $4, to_timestamp($5))`
		_, err := db.Exec(query, data.SensorID, data.Temp, data.Humidity, data.Status, data.Timestamp)
		if err != nil {
			log.Printf("Error inserting telemetry: %v\n", err)
		} else {
			log.Printf("Saved telemetry for sensor: %s\n", data.SensorID)
		}
	})

	client := mqtt.NewClient(opts)
	if token := client.Connect(); token.Wait() && token.Error() != nil {
		log.Fatal(token.Error())
	}
	defer client.Disconnect(250)

	log.Printf("Connected to MQTT broker at %s\n", mqttURL)

	// 3. Subscribe to telemetry topic
	// Topic structure: sensors/+/telemetry
	topic := "sensors/+/telemetry"
	if token := client.Subscribe(topic, 1, nil); token.Wait() && token.Error() != nil {
		log.Fatalf("Error subscribing to topic: %v\n", token.Error())
	}
	log.Printf("Subscribed to topic: %s\n", topic)

	// Keep alive
	quit := make(chan os.Signal, 1)
	signal.Notify(quit, syscall.SIGINT, syscall.SIGTERM)
	<-quit
	log.Println("Shutting down telemetry-service...")
}

func getEnv(key, defaultValue string) string {
	value := os.Getenv(key)
	if value == "" {
		return defaultValue
	}
	return value
}
