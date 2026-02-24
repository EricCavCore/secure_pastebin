/*********************************
 *  File     : main.go
 *  Purpose  : Backend entry point
 *  Authors  : Eric Caverly
 */

package main

import (
	"log"
	"net/http"
	"os"
	"time"

	"github.com/redis/go-redis/v9"
)

const (
	max_days            = 15
	max_note_size_bytes = 1024 * 30 // 30 kB
	max_num_links       = 500
	max_max_clicks      = 100
)

var rc *redis.Client

func main() {
	redis_pass := os.Getenv("REDIS_PASSWORD")

	rc = redis.NewClient(&redis.Options{
		// Addr:     "db.spb.arpa:6379",
		Addr:     "localhost:6379",
		Password: redis_pass,
		DB:       0,
		Protocol: 3,
	})

	// Create the HTTP server
	r := http.NewServeMux()

	fs := http.FileServer(http.Dir("./www"))
	r.Handle("/", http.StripPrefix("/", fs))

	r.HandleFunc("GET /api/note/{id}", web_get_note)
	r.HandleFunc("POST /api/note", web_post_note)
	r.HandleFunc("GET /api/health", web_health_check)

	srv := &http.Server{
		Addr:         ":8080",
		Handler:      r,
		ReadTimeout:  5 * time.Second,
		WriteTimeout: 10 * time.Second,
		IdleTimeout:  120 * time.Second,
	}

	log.Printf("Started listening on %s\n", srv.Addr)

	if err := srv.ListenAndServe(); err != nil {
		log.Fatalf("Web server crashed: %s\n", err.Error())
	}
}
