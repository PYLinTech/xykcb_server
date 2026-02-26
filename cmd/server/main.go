package main

import (
	"fmt"
	"log"
	"net/http"
	"os"
	"xykcb_server/internal/config"
	"xykcb_server/internal/handler"

	// Import school providers to trigger init registration
	_ "xykcb_server/internal/provider/schools"
)

func main() {
	// Load configuration
	cfg := config.Load()

	// Read 404.html content
	notFoundContent, err := os.ReadFile("assets/404.html")
	if err != nil {
		log.Fatalf("Failed to read 404.html: %v", err)
	}

	// Create handler
	courseHandler := handler.NewCourseHandler()

	// Set up router
	mux := http.NewServeMux()
	mux.HandleFunc("/api/get-course-data", courseHandler.HandleCourse)
	mux.HandleFunc("/api/get-support-school", courseHandler.GetSupportedSchools)

	// Default handler for undefined routes - return 404.html content
	mux.HandleFunc("/", func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusNotFound)
		w.Header().Set("Content-Type", "text/html; charset=utf-8")
		w.Write(notFoundContent)
	})

	// Configure server with timeouts
	srv := &http.Server{
		Addr:         ":" + cfg.Server.Port,
		Handler:      mux,
		ReadTimeout:  cfg.Server.ReadTimeout,
		WriteTimeout: cfg.Server.WriteTimeout,
	}

	log.Printf("Server starting on port %s", cfg.Server.Port)
	log.Fatal(srv.ListenAndServe())

	// Keep compiler happy with fmt
	_ = fmt.Sprintf("%v", cfg)
}
