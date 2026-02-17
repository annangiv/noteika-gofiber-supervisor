package main

import (
	"bytes"
	"context"
	"fmt"
	"io"
	"log"
	"math/rand"
	"net/http"
	"os"
	"os/signal"
	"sync"
	"syscall"
	"time"

	"my-app/supervisor"

	"github.com/gofiber/fiber/v2"
	"github.com/gofiber/fiber/v2/middleware/recover"
)

// Simple health checker
type HealthChecker struct {
	targets []string
	mutex   sync.RWMutex
}

func NewHealthChecker(targets []string) *HealthChecker {
	hc := &HealthChecker{targets: targets}
	go hc.startHealthChecks()
	return hc
}

func (hc *HealthChecker) GetHealthyTargets() []string {
	hc.mutex.RLock()
	defer hc.mutex.RUnlock()

	// For now, return all targets
	return hc.targets
}

func (hc *HealthChecker) startHealthChecks() {
	ticker := time.NewTicker(10 * time.Second)
	defer ticker.Stop()

	for range ticker.C {
		hc.checkHealth()
	}
}

func (hc *HealthChecker) checkHealth() {
	hc.mutex.Lock()
	defer hc.mutex.Unlock()

	// For now, we'll assume all targets are healthy
	log.Println("Performing health checks on targets:", hc.targets)
}

func main() {
	// Create supervisor
	sup := supervisor.NewSupervisor()
	defer sup.Stop()

	// Define backend server ports
	backendPorts := []int{3001, 3002, 3003}
	backendURLs := make([]string, len(backendPorts))

	// Start backend servers
	for i, port := range backendPorts {
		workerID := fmt.Sprintf("backend-server-%d", port)
		backendURLs[i] = fmt.Sprintf("http://localhost:%d", port)
		sup.AddWorker(workerID, createBackendServerWorker(port))
	}

	// Create health checker
	healthChecker := NewHealthChecker(backendURLs)

	// Give backend servers time to start
	time.Sleep(2 * time.Second)

	// Start load balancer (frontend server)
	go startLoadBalancerServer(backendURLs, healthChecker)

	log.Println("Load balancer started on :8080")
	log.Println("Backend servers:", backendURLs)

	// Graceful shutdown handling
	quit := make(chan os.Signal, 1)
	signal.Notify(quit, syscall.SIGINT, syscall.SIGTERM)
	<-quit

	log.Println("Shutting down servers...")
	sup.Stop()
}

func createBackendServerWorker(port int) func(ctx context.Context) error {
	return func(ctx context.Context) error {
		app := fiber.New(fiber.Config{
			ErrorHandler: func(c *fiber.Ctx, err error) error {
				log.Printf("Backend server error on port %d: %v", port, err)
				return c.Status(500).JSON(fiber.Map{"error": "Internal Server Error"})
			},
		})

		app.Use(recover.New())

		// Backend routes
		app.Get("/health", func(c *fiber.Ctx) error {
			return c.JSON(fiber.Map{
				"status": "healthy",
				"port":   port,
				"time":   time.Now().Format(time.RFC3339),
			})
		})

		app.Get("/api/data", func(c *fiber.Ctx) error {
			// Simulatign some work with slight variation per server
			time.Sleep(time.Duration(rand.Intn(100)) * time.Millisecond)
			return c.JSON(fiber.Map{
				"data":       []string{"item1", "item2", "item3"},
				"server":     port,
				"timestamp":  time.Now().Unix(),
				"request_id": fmt.Sprintf("%d-%d", port, time.Now().UnixNano()),
			})
		})

		app.Post("/api/data", func(c *fiber.Ctx) error {
			var payload map[string]interface{}
			if err := c.BodyParser(&payload); err != nil {
				return c.Status(400).JSON(fiber.Map{"error": "Invalid JSON"})
			}

			return c.JSON(fiber.Map{
				"message":    "Data received",
				"server":     port,
				"received":   payload,
				"request_id": fmt.Sprintf("%d-%d", port, time.Now().UnixNano()),
			})
		})

		serverAddr := fmt.Sprintf(":%d", port)
		log.Printf("Starting backend server on %s", serverAddr)

		go func() {
			if err := app.Listen(serverAddr); err != nil {
				log.Printf("Backend server error on port %d: %v", port, err)
			}
		}()

		// Wait for context cancellation
		<-ctx.Done()
		log.Printf("Shutting down backend server on port %d", port)
		return app.Shutdown()
	}
}

func startLoadBalancerServer(backendURLs []string, healthChecker *HealthChecker) {
	app := fiber.New(fiber.Config{
		ErrorHandler: func(c *fiber.Ctx, err error) error {
			log.Printf("Load balancer error: %v", err)
			return c.Status(503).JSON(fiber.Map{"error": "Service Unavailable"})
		},
	})

	app.Use(recover.New())

	// Health check endpoint for load balancer itself
	app.Get("/health", func(c *fiber.Ctx) error {
		return c.JSON(fiber.Map{
			"status":  "load balancer healthy",
			"servers": backendURLs,
			"time":    time.Now().Format(time.RFC3339),
		})
	})

	// Counter for round-robin
	var counter int
	var counterMutex sync.Mutex

	// Proxy all requests to backend servers
	app.Use(func(c *fiber.Ctx) error {
		// Get healthy targets
		healthyTargets := healthChecker.GetHealthyTargets()
		if len(healthyTargets) == 0 {
			return c.Status(503).JSON(fiber.Map{
				"error": "No healthy backend servers available",
			})
		}

		// Round-robin selection
		counterMutex.Lock()
		selectedTarget := healthyTargets[counter%len(healthyTargets)]
		counter++
		counterMutex.Unlock()

		log.Printf("Routing request to: %s%s", selectedTarget, c.OriginalURL())

		// Forward request to selected backend
		return forwardRequest(c, selectedTarget)
	})

	log.Fatal(app.Listen(":8080"))
}

func forwardRequest(c *fiber.Ctx, targetURL string) error {
	// Construct full URL
	fullURL := fmt.Sprintf("%s%s", targetURL, c.OriginalURL())

	// Get request body as bytes
	bodyBytes := c.Request().Body()

	// Create HTTP client
	client := &http.Client{
		Timeout: 30 * time.Second,
	}

	// Create request with body - convert []byte to io.Reader
	var bodyReader io.Reader
	if len(bodyBytes) > 0 {
		bodyReader = bytes.NewReader(bodyBytes)
	} else {
		bodyReader = nil
	}

	// Create request
	req, err := http.NewRequest(c.Method(), fullURL, bodyReader)
	if err != nil {
		return err
	}

	// Copy headers
	c.Request().Header.VisitAll(func(key, value []byte) {
		req.Header.Set(string(key), string(value))
	})

	// Make request
	resp, err := client.Do(req)
	if err != nil {
		log.Printf("Backend request failed: %v", err)
		return c.Status(502).JSON(fiber.Map{
			"error": "Backend request failed",
		})
	}
	defer resp.Body.Close()

	// Copy response headers
	for key, values := range resp.Header {
		for _, value := range values {
			c.Response().Header.Set(key, value)
		}
	}

	// Set status code
	c.Response().SetStatusCode(resp.StatusCode)

	// Copy response body
	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return err
	}
	c.Response().SetBody(body)

	return nil
}
