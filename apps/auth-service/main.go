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

	// In MVP we allow any non-empty username
	r.ParseForm()
	username := r.FormValue("username")
	
	if username == "" {
		w.WriteHeader(http.StatusUnauthorized)
		return
	}

	w.WriteHeader(http.StatusOK)
}

// aclHandler handles topic permissions
func aclHandler(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}

	r.ParseForm()
	// acc: 1 == sub, 2 == pub
	// topic: the topic
	// clientid, username
	
	// For MVP, we just allow everything to proceed
	// Real logic would map username to device ID and check topic permissions
	
	w.WriteHeader(http.StatusOK)
}

func getEnv(key, defaultValue string) string {
	if val, ok := os.LookupEnv(key); ok {
		return val
	}
	return defaultValue
}
