package main

import (
	"bytes"
	"encoding/json"
	"log"
	"net/http"
	"os"
	"os/signal"
	"syscall"

	mqtt "github.com/eclipse/paho.mqtt.golang"
)

type SensorData struct {
	SensorID  string  `json:"sensor_id"`
	Temp      float64 `json:"temperature"`
	Humidity  float64 `json:"humidity"`
	Status    string  `json:"status"`
	Timestamp int64   `json:"timestamp"`
}

func main() {
	mqttURL := getEnv("MQTT_URL", "tcp://mosquitto:1883")
	commandURL := getEnv("COMMAND_SERVICE_URL", "http://command-service:8080/api/v1/commands")

	opts := mqtt.NewClientOptions().AddBroker(mqttURL)
	opts.SetClientID("scenario_service")
	opts.SetCleanSession(true)

	opts.SetDefaultPublishHandler(func(client mqtt.Client, msg mqtt.Message) {
		var data SensorData
		if err := json.Unmarshal(msg.Payload(), &data); err != nil {
			log.Printf("Error parsing telemetry: %v", err)
			return
		}

		log.Printf("Received telemetry: %s temp=%.2f", data.SensorID, data.Temp)

		// Basic scenario rule: turn on heating if temp < 22
		if data.Temp < 22.0 {
			triggerCommand(commandURL, data.SensorID, "turn_on_heating")
		}
	})

	client := mqtt.NewClient(opts)
	if token := client.Connect(); token.Wait() && token.Error() != nil {
		log.Fatalf("MQTT connect error: %v", token.Error())
	}
	defer client.Disconnect(250)

	topic := "sensors/+/telemetry"
	if token := client.Subscribe(topic, 1, nil); token.Wait() && token.Error() != nil {
		log.Fatalf("MQTT subscribe error: %v", token.Error())
	}
	log.Printf("Scenario Service started. Subscribed to %s", topic)

	quit := make(chan os.Signal, 1)
	signal.Notify(quit, syscall.SIGINT, syscall.SIGTERM)
	<-quit
	log.Println("Shutting down...")
}

func triggerCommand(url string, deviceID string, action string) {
	payload, _ := json.Marshal(map[string]string{
		"device_id": deviceID,
		"action":    action,
	})

	resp, err := http.Post(url, "application/json", bytes.NewBuffer(payload))
	if err != nil {
		log.Printf("Failed to trigger command: %v", err)
		return
	}
	defer resp.Body.Close()
	
	log.Printf("Triggered command %s for device %s. Status: %d", action, deviceID, resp.StatusCode)
}

func getEnv(key, defaultValue string) string {
	if val, ok := os.LookupEnv(key); ok {
		return val
	}
	return defaultValue
}
