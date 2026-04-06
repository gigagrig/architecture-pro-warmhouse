package main

import (
	"log"
	"net/http"
	"net/http/httputil"
	"net/url"
	"os"
	"strings"
)

func main() {
	deviceSvcURL := getEnv("DEVICE_SERVICE_URL", "http://device-service:8080")
	commandSvcURL := getEnv("COMMAND_SERVICE_URL", "http://command-service:8080")
	smartHomeURL := getEnv("SMART_HOME_URL", "http://app:8080")

	deviceProxy := createProxy(deviceSvcURL)
	commandProxy := createProxy(commandSvcURL)
	smartHomeProxy := createProxy(smartHomeURL)

	http.HandleFunc("/", func(w http.ResponseWriter, r *http.Request) {
		path := r.URL.Path

		log.Printf("Gateway received request for: %s", path)

		if strings.HasPrefix(path, "/api/v2/devices") {
			deviceProxy.ServeHTTP(w, r)
			return
		}

		if strings.HasPrefix(path, "/api/v2/commands") {
			commandProxy.ServeHTTP(w, r)
			return
		}

		// Default to smart_home monolith
		smartHomeProxy.ServeHTTP(w, r)
	})

	port := getEnv("PORT", "8080")
	log.Printf("API Gateway starting on port %s", port)
	if err := http.ListenAndServe(":"+port, nil); err != nil {
		log.Fatalf("Gateway failed: %v", err)
	}
}

func createProxy(target string) *httputil.ReverseProxy {
	targetURL, err := url.Parse(target)
	if err != nil {
		log.Fatalf("Invalid target URL %s: %v", target, err)
	}
	proxy := httputil.NewSingleHostReverseProxy(targetURL)
	return proxy
}

func getEnv(key, defaultValue string) string {
	if val, ok := os.LookupEnv(key); ok {
		return val
	}
	return defaultValue
}
