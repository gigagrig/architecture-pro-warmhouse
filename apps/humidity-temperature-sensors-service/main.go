package main

import (
	"encoding/json"
	"fmt"
	"log"
	"math/rand"
	"time"

	mqtt "github.com/eclipse/paho.mqtt.golang"
)

// SensorData represents the telemetry payload
type SensorData struct {
	SensorID  string  `json:"sensor_id"`
	Temp      float64 `json:"temperature"`
	Humidity  float64 `json:"humidity"`
	Status    string  `json:"status"`
	Timestamp int64   `json:"timestamp"`
}

func main() {
	// 1. Broker Configuration
	opts := mqtt.NewClientOptions().AddBroker("tcp://localhost:1883")
	opts.SetClientID("go_sensor_producer")
	opts.SetCleanSession(true)

	client := mqtt.NewClient(opts)
	if token := client.Connect(); token.Wait() && token.Error() != nil {
		log.Fatal(token.Error())
	}
	defer client.Disconnect(250)

	fmt.Println("Producer connected to broker. Sending telemetry...")

	// 2. Simulation Loop
	for {
		data := SensorData{
			SensorID:  "DHT22-Room101",
			Temp:      20.0 + rand.Float64()*10.0,
			Humidity:  40.0 + rand.Float64()*20.0,
			Status:    "active",
			Timestamp: time.Now().Unix(),
		}

		payload, _ := json.Marshal(data)

		// 3. Publish with QoS 1 (At least once delivery)
		// Topic structure: sensors/<id>/telemetry
		token := client.Publish("sensors/DHT22-Room101/telemetry", 1, true, payload)
		token.Wait()

		fmt.Printf("Published: %s\n", payload)
		time.Sleep(5 * time.Second)
	}
}
