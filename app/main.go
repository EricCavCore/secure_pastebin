/*********************************
 *  File     : main.go
 *  Purpose  : Backend entry point
 *  Authors  : Eric Caverly
 */

package main

import (
	"log"
	"net/http"
	"time"

	"github.com/redis/go-redis/v9"
)

const (
	max_days            = 15
	max_note_size_bytes = 1024 * 30 // 30 kB
)

var rc = redis.NewClient(&redis.Options{
	// Addr:     "db.spb.arpa:6379",
	Addr:     "localhost:6379",
	Password: "",
	DB:       0,
	Protocol: 2,
})

func main() {
	// Create the HTTP server
	r := http.NewServeMux()

	r.Handle("/", http.FileServer(http.Dir("./www")))

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
