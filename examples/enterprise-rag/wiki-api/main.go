package main

import (
	"encoding/json"
	"log"
	"net/http"
	"os"
	"time"
)

type WikiPage struct {
	ID          string    `json:"id"`
	Title       string    `json:"title"`
	Content     string    `json:"content"`
	Author      string    `json:"author"`
	Category    string    `json:"category"`
	PublishedAt time.Time `json:"published_at"`
	UpdatedAt   time.Time `json:"updated_at"`
}

var wikiPages = []WikiPage{
	{
		ID:          "wiki-001",
		Title:       "Architecture Overview",
		Content:     "Our system follows a microservices architecture with API gateway pattern. Services communicate via gRPC for internal calls and REST for external APIs. We use event-driven architecture for asynchronous processing. Data is stored in PostgreSQL for transactional data and Redis for caching. Message queue uses RabbitMQ for reliable message delivery.",
		Author:      "Architecture Team",
		Category:    "Technical",
		PublishedAt: time.Now().AddDate(0, -2, 0),
		UpdatedAt:   time.Now().AddDate(0, -1, 0),
	},
	{
		ID:          "wiki-002",
		Title:       "Service Mesh Configuration",
		Content:     "All services are registered in the service mesh for service discovery and load balancing. Circuit breakers are configured with 50% failure threshold and 30-second timeout. Retry policies: 3 attempts with exponential backoff. Health checks run every 10 seconds. Traffic splitting is configured for canary deployments.",
		Author:      "Platform Team",
		Category:    "Infrastructure",
		PublishedAt: time.Now().AddDate(0, -3, 0),
		UpdatedAt:   time.Now().AddDate(0, -2, 0),
	},
	{
		ID:          "wiki-003",
		Title:       "CI/CD Pipeline",
		Content:     "Continuous integration runs on every commit: unit tests, integration tests, static analysis, and security scanning. Deployment pipeline: build Docker image, push to registry, deploy to staging, run smoke tests, deploy to production with blue-green strategy. Rollback is automated if health checks fail. All deployments are logged and audited.",
		Author:      "DevOps Team",
		Category:    "Development",
		PublishedAt: time.Now().AddDate(0, -1, 0),
		UpdatedAt:   time.Now().AddDate(0, 0, -5),
	},
	{
		ID:          "wiki-004",
		Title:       "Disaster Recovery Plan",
		Content:     "RTO (Recovery Time Objective): 4 hours for critical systems, 24 hours for non-critical. RPO (Recovery Point Objective): 1 hour for critical data. Failover is automated to secondary data center. DR drills are conducted quarterly. Backup systems are tested monthly. Contact list includes: CTO, CISO, Operations Manager, and on-call engineers.",
		Author:      "Operations Team",
		Category:    "Operations",
		PublishedAt: time.Now().AddDate(0, -6, 0),
		UpdatedAt:   time.Now().AddDate(0, -3, 0),
	},
	{
		ID:          "wiki-005",
		Title:       "Performance Optimization Guidelines",
		Content:     "Database queries must use indexes and avoid N+1 queries. API responses should be cached for at least 5 minutes for static data. Connection pooling: max 100 connections per service. Timeout settings: 30s for external APIs, 5s for internal services. Use CDN for static assets. Implement pagination for large result sets. Monitor p95 and p99 latency metrics.",
		Author:      "Performance Team",
		Category:    "Technical",
		PublishedAt: time.Now().AddDate(0, -4, 0),
		UpdatedAt:   time.Now().AddDate(0, -2, 0),
	},
}

func healthHandler(w http.ResponseWriter, r *http.Request) {
	w.WriteHeader(http.StatusOK)
	json.NewEncoder(w).Encode(map[string]string{"status": "healthy"})
}

func getPagesHandler(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(wikiPages)
}

func main() {
	port := os.Getenv("PORT")
	if port == "" {
		port = "8080"
	}

	mux := http.NewServeMux()
	mux.HandleFunc("/health", healthHandler)
	mux.HandleFunc("/api/pages", getPagesHandler)

	server := &http.Server{
		Addr:         ":" + port,
		Handler:      mux,
		ReadTimeout:  15 * time.Second,
		WriteTimeout: 15 * time.Second,
		IdleTimeout:  60 * time.Second,
	}

	log.Printf("Wiki API server starting on port %s", port)
	log.Fatal(server.ListenAndServe())
}

