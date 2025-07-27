package main

import (
	"fmt"
	"net/http"
	"time"
)

// Simple test to verify what endpoints are actually available on the server
func main() {
	baseURL := "http://localhost:8080"

	// Test endpoints we expect to exist
	endpoints := []string{
		"/api/v1/conversations/test-conversation-1/graph",        // Clean API
		"/api/v1/conversations/test-conversation-1",              // Clean API
		"/api/ui/graph-data?conversation_id=test-conversation-1", // Old API
		"/api/chat", // WebBFF
		"/health",   // Health check
	}

	client := &http.Client{Timeout: 5 * time.Second}

	fmt.Println("🔍 Testing API endpoints on", baseURL)
	fmt.Println("==================================================")

	for _, endpoint := range endpoints {
		resp, err := client.Get(baseURL + endpoint)
		if err != nil {
			fmt.Printf("❌ %s - Connection failed: %v\n", endpoint, err)
			continue
		}
		defer resp.Body.Close()

		status := resp.StatusCode
		switch {
		case status == 200:
			fmt.Printf("✅ %s - OK (%d)\n", endpoint, status)
		case status == 404:
			fmt.Printf("🔍 %s - Not Found (%d)\n", endpoint, status)
		case status >= 400 && status < 500:
			fmt.Printf("⚠️  %s - Client Error (%d)\n", endpoint, status)
		case status >= 500:
			fmt.Printf("💥 %s - Server Error (%d)\n", endpoint, status)
		default:
			fmt.Printf("❓ %s - Unexpected (%d)\n", endpoint, status)
		}
	}

	fmt.Println("==================================================")
	fmt.Println("🎯 This test shows which API endpoints are actually wired up to the server")
}
