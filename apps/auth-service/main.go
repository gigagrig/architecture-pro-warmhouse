package main

import (
	"log"
	"net/http"
	"os"
)

func main() {
	mux := http.NewServeMux()
	mux.HandleFunc("/auth", authHandler)
	mux.HandleFunc("/acl", aclHandler)

	port := getEnv("PORT", "8080")
	log.Printf("Auth & ACL Service starting on port %s", port)
	if err := http.ListenAndServe(":"+port, mux); err != nil {
		log.Fatalf("Server failed: %v", err)
	}
}

// mosquitto-go-auth sends username and password
func authHandler(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}

	r.ParseForm()
	username := r.FormValue("username")

	if username == "" {
		w.WriteHeader(http.StatusUnauthorized)
		return
	}

	deviceSvcURL := getEnv("DEVICE_SERVICE_URL", "http://device-service:8080")
	resp, err := http.Get(deviceSvcURL + "/api/v2/devices/" + username)
	if err != nil || resp.StatusCode != http.StatusOK {
		log.Printf("Auth failed for username/device_id: %s", username)
		w.WriteHeader(http.StatusUnauthorized)
		return
	}
	if resp != nil {
		resp.Body.Close()
	}

	log.Printf("Auth succeeded for username: %s", username)
	w.WriteHeader(http.StatusOK)
}

// aclHandler handles topic permissions
func aclHandler(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}

	r.ParseForm()
	username := r.FormValue("username")
	topic := r.FormValue("topic")
	acc := r.FormValue("acc") // 1 == sub, 2 == pub

	log.Printf("ACL check: user=%s topic=%s acc=%s", username, topic, acc)

	deviceSvcURL := getEnv("DEVICE_SERVICE_URL", "http://device-service:8080")
	resp, err := http.Get(deviceSvcURL + "/api/v2/devices/" + username)
	if err != nil || resp.StatusCode != http.StatusOK {
		log.Printf("ACL denied for device/user: %s", username)
		w.WriteHeader(http.StatusUnauthorized)
		return
	}
	if resp != nil {
		resp.Body.Close()
	}

	w.WriteHeader(http.StatusOK)
}

func getEnv(key, defaultValue string) string {
	if val, ok := os.LookupEnv(key); ok {
		return val
	}
	return defaultValue
}
