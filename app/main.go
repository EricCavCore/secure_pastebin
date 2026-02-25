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
	"strings"
	"time"

	"github.com/redis/go-redis/v9"
)

const (
	max_days            = 15
	max_note_size_bytes = 1024 * 30 // 30 kB
	max_num_links       = 500
	max_max_clicks      = 100
)

var (
	rc             *redis.Client
	trustedProxies []string
	getLimiter     *RateLimiter
	postLimiter    *RateLimiter
)

func main() {
	redis_pass := os.Getenv("REDIS_PASSWORD")
	redis_addr := os.Getenv("REDIS_ADDR")

	if redis_addr == "" {
		redis_addr = "localhost:6379"
	}

	// Parse trusted proxy CIDR ranges from environment
	tp := os.Getenv("TRUSTED_PROXIES")
	if tp != "" {
		for _, cidr := range strings.Split(tp, ",") {
			trimmed := strings.TrimSpace(cidr)
			if trimmed != "" {
				trustedProxies = append(trustedProxies, trimmed)
			}
		}
		log.Printf("Trusted proxies: %v\n", trustedProxies)
	}

	rc = redis.NewClient(&redis.Options{
		Addr:     redis_addr,
		Password: redis_pass,
		DB:       0,
		Protocol: 3,
	})

	// Initialize rate limiters
	getLimiter = NewRateLimiter(10, 20)  // 10 req/s, burst of 20
	postLimiter = NewRateLimiter(5, 10)  // 5 req/s, burst of 10

	// Create the HTTP server
	r := http.NewServeMux()

	fs := http.FileServer(http.Dir("./www"))
	r.Handle("/", http.StripPrefix("/", fs))

	r.HandleFunc("GET /api/note/{id}", rateLimitedHandler(getLimiter, web_get_note))
	r.HandleFunc("POST /api/note/{id}/verify", rateLimitedHandler(getLimiter, web_verify_note))
	r.HandleFunc("POST /api/note", rateLimitedHandler(postLimiter, web_post_note))
	r.HandleFunc("GET /api/health", web_health_check)

	// Wrap mux with security headers and CORS middleware
	handler := securityHeaders(corsMiddleware(r))

	srv := &http.Server{
		Addr:         ":8080",
		Handler:      handler,
		ReadTimeout:  5 * time.Second,
		WriteTimeout: 10 * time.Second,
		IdleTimeout:  120 * time.Second,
	}

	log.Printf("Started listening on %s\n", srv.Addr)

	if err := srv.ListenAndServe(); err != nil {
		log.Fatalf("Web server crashed: %s\n", err.Error())
	}
}
