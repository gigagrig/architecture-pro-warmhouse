package main

import (
	"encoding/json"
	"fmt"
	"log"
	"net/http"
	"os"

	mqtt "github.com/eclipse/paho.mqtt.golang"
)

var mqttClient mqtt.Client

func main() {
	mqttURL := getEnv("MQTT_URL", "tcp://mosquitto:1883")
	opts := mqtt.NewClientOptions().AddBroker(mqttURL)
	opts.SetClientID("command_service")
	opts.SetCleanSession(true)

	mqttClient = mqtt.NewClient(opts)
	if token := mqttClient.Connect(); token.Wait() && token.Error() != nil {
		log.Fatalf("Failed to connect to MQTT: %v", token.Error())
	}
	defer mqttClient.Disconnect(250)
	log.Printf("Connected to MQTT broker at %s", mqttURL)

	mux := http.NewServeMux()
	mux.HandleFunc("/api/v2/commands", commandHandler)

	port := getEnv("PORT", "8080")
	log.Printf("Command Service starting on port %s", port)
	if err := http.ListenAndServe(":"+port, mux); err != nil {
		log.Fatalf("Server failed: %v", err)
	}
}

func commandHandler(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		http.Error(w, `{"error":"Method not allowed"}`, http.StatusMethodNotAllowed)
		return
	}

	var req struct {
		DeviceID string `json:"device_id"`
		Action   string `json:"action"`
	}

	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		http.Error(w, `{"error":"Invalid body"}`, http.StatusBadRequest)
		return
	}

	if req.DeviceID == "" || req.Action == "" {
		http.Error(w, `{"error":"Missing fields"}`, http.StatusBadRequest)
		return
	}

	deviceSvcURL := getEnv("DEVICE_SERVICE_URL", "http://device-service:8080")
	resp, err := http.Get(fmt.Sprintf("%s/api/v2/devices/%s", deviceSvcURL, req.DeviceID))
	if err != nil {
		log.Printf("Failed to contact device-service: %v", err)
		http.Error(w, `{"error":"Internal error"}`, http.StatusInternalServerError)
		return
	}
	if resp.StatusCode != http.StatusOK {
		resp.Body.Close()
		http.Error(w, `{"error":"Device not found"}`, http.StatusBadRequest)
		return
	}

	var device struct {
		Protocol string `json:"protocol"`
	}
	if err := json.NewDecoder(resp.Body).Decode(&device); err != nil {
		resp.Body.Close()
		http.Error(w, `{"error":"Failed to parse device"}`, http.StatusInternalServerError)
		return
	}
	resp.Body.Close()

	if device.Protocol == "mqtt" {
		topic := fmt.Sprintf("devices/%s/commands", req.DeviceID)
		payload, _ := json.Marshal(map[string]string{"action": req.Action})

		token := mqttClient.Publish(topic, 1, false, payload)
		token.Wait()
		if err := token.Error(); err != nil {
			log.Printf("MQTT publish error: %v", err)
			http.Error(w, `{"error":"Failed to send command"}`, http.StatusInternalServerError)
			return
		}
		log.Printf("Sent MQTT command %s to %s", string(payload), topic)
	} else if device.Protocol == "http" {
		log.Printf("Sent mock HTTP POST command %s to device %s", req.Action, req.DeviceID)
	} else {
		log.Printf("Unsupported protocol %s for device %s", device.Protocol, req.DeviceID)
		http.Error(w, `{"error":"Unsupported protocol"}`, http.StatusBadRequest)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusOK)
	w.Write([]byte(`{"status":"command_sent"}`))
}

func getEnv(key, defaultValue string) string {
	if val, ok := os.LookupEnv(key); ok {
		return val
	}
	return defaultValue
}
