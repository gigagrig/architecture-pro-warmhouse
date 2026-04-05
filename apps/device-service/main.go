package main

import (
	"database/sql"
	"encoding/json"
	"log"
	"net/http"
	"os"
	"time"

	"github.com/google/uuid"
	_ "github.com/lib/pq"
)

type Device struct {
	ID        string    `json:"id"`
	UserID    string    `json:"user_id"`
	Name      string    `json:"name"`
	Protocol  string    `json:"protocol"`
	CreatedAt time.Time `json:"created_at"`
}

var db *sql.DB

func main() {
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
		log.Printf("Waiting for DB... %v", err)
		time.Sleep(2 * time.Second)
	}
	if err != nil {
		log.Fatalf("Unable to connect to database: %v", err)
	}
	defer db.Close()
	log.Println("Connected to DB successfully")

	mux := http.NewServeMux()
	mux.HandleFunc("/api/v1/devices", func(w http.ResponseWriter, r *http.Request) {
		devicesHandler(w, r)
	})
	mux.HandleFunc("/api/v1/devices/", func(w http.ResponseWriter, r *http.Request) {
		deviceHandler(w, r)
	})

	port := getEnv("PORT", "8080")
	log.Printf("Device Service starting on port %s", port)
	if err := http.ListenAndServe(":"+port, mux); err != nil {
		log.Fatalf("Server failed: %v", err)
	}
}

func deviceHandler(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "application/json")
	if r.Method != http.MethodGet {
		http.Error(w, `{"error":"Method not allowed"}`, http.StatusMethodNotAllowed)
		return
	}

	id := r.URL.Path[len("/api/v1/devices/"):]
	if id == "" {
		http.Error(w, `{"error":"Missing ID"}`, http.StatusBadRequest)
		return
	}

	var d Device
	err := db.QueryRow("SELECT id, user_id, name, protocol, created_at FROM devices WHERE id = $1", id).Scan(&d.ID, &d.UserID, &d.Name, &d.Protocol, &d.CreatedAt)
	if err != nil {
		if err == sql.ErrNoRows {
			http.Error(w, `{"error":"Device not found"}`, http.StatusNotFound)
		} else {
			http.Error(w, `{"error":"DB error"}`, http.StatusInternalServerError)
		}
		return
	}

	json.NewEncoder(w).Encode(d)
}

func devicesHandler(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "application/json")

	if r.Method == http.MethodGet {
		rows, err := db.Query("SELECT id, user_id, name, protocol, created_at FROM devices ORDER BY created_at DESC")
		if err != nil {
			http.Error(w, `{"error":"DB error"}`, http.StatusInternalServerError)
			return
		}
		defer rows.Close()

		devices := []Device{}
		for rows.Next() {
			var d Device
			if err := rows.Scan(&d.ID, &d.UserID, &d.Name, &d.Protocol, &d.CreatedAt); err != nil {
				log.Printf("Scan error: %v", err)
				continue
			}
			devices = append(devices, d)
		}
		json.NewEncoder(w).Encode(devices)
		return
	}

	if r.Method == http.MethodPost {
		var req struct {
			Name     string `json:"name"`
			Protocol string `json:"protocol"`
			UserID   string `json:"user_id"`
		}
		if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
			http.Error(w, `{"error":"Invalid body"}`, http.StatusBadRequest)
			return
		}
		if req.UserID == "" {
			req.UserID = "default-user" // Mock user id
		}
		if req.Protocol == "" {
			req.Protocol = "mqtt"
		}

		id := uuid.New().String()
		_, err := db.Exec("INSERT INTO devices (id, user_id, name, protocol) VALUES ($1, $2, $3, $4)",
			id, req.UserID, req.Name, req.Protocol)
		if err != nil {
			http.Error(w, `{"error":"Insert failed"}`, http.StatusInternalServerError)
			return
		}

		d := Device{ID: id, UserID: req.UserID, Name: req.Name, Protocol: req.Protocol, CreatedAt: time.Now()}
		w.WriteHeader(http.StatusCreated)
		json.NewEncoder(w).Encode(d)
		return
	}

	http.Error(w, `{"error":"Method not allowed"}`, http.StatusMethodNotAllowed)
}

func getEnv(key, defaultValue string) string {
	if val, ok := os.LookupEnv(key); ok {
		return val
	}
	return defaultValue
}
