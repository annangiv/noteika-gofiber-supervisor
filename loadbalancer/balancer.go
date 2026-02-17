package loadbalancer

import (
	"fmt"
	"log"
	"net/http"
	"sync"
	"time"
)

type Server struct {
	URL       string
	Healthy   bool
	LastCheck time.Time
	mutex     sync.RWMutex
}

type LoadBalancer struct {
	servers      []*Server
	currentIndex int
	mutex        sync.RWMutex
}

func NewLoadBalancer(serverURLs []string) *LoadBalancer {
	lb := &LoadBalancer{
		servers:      make([]*Server, len(serverURLs)),
		currentIndex: 0,
	}

	for i, url := range serverURLs {
		lb.servers[i] = &Server{
			URL:     url,
			Healthy: true, // Assuming healthy initially
		}
	}

	// Start health checking
	go lb.startHealthChecks()

	return lb
}

func (lb *LoadBalancer) GetNextServer() (*Server, error) {
	lb.mutex.Lock()
	defer lb.mutex.Unlock()

	// Try up to len(servers) times to find a healthy server
	for i := 0; i < len(lb.servers); i++ {
		server := lb.servers[lb.currentIndex]
		lb.currentIndex = (lb.currentIndex + 1) % len(lb.servers)

		if server.IsHealthy() {
			return server, nil
		}
	}

	return nil, fmt.Errorf("no healthy servers available")
}

func (s *Server) IsHealthy() bool {
	s.mutex.RLock()
	defer s.mutex.RUnlock()
	return s.Healthy
}

func (s *Server) SetHealth(healthy bool) {
	s.mutex.Lock()
	defer s.mutex.Unlock()
	s.Healthy = healthy
	s.LastCheck = time.Now()
}

func (lb *LoadBalancer) startHealthChecks() {
	ticker := time.NewTicker(5 * time.Second)
	defer ticker.Stop()

	for range ticker.C {
		lb.checkAllServers()
	}
}

func (lb *LoadBalancer) checkAllServers() {
	for _, server := range lb.servers {
		go func(s *Server) {
			healthy := lb.checkServerHealth(s.URL)
			s.SetHealth(healthy)
			if healthy {
				log.Printf("Server %s is healthy", s.URL)
			} else {
				log.Printf("Server %s is unhealthy", s.URL)
			}
		}(server)
	}
}

func (lb *LoadBalancer) checkServerHealth(url string) bool {
	healthURL := fmt.Sprintf("%s/health", url)
	client := &http.Client{
		Timeout: 3 * time.Second,
	}

	resp, err := client.Get(healthURL)
	if err != nil {
		return false
	}
	defer resp.Body.Close()

	return resp.StatusCode == 200
}

// Round-robin proxy function
func (lb *LoadBalancer) ProxyRequest(targetURL string) (*http.Response, error) {
	server, err := lb.GetNextServer()
	if err != nil {
		return nil, err
	}

	fullURL := fmt.Sprintf("%s%s", server.URL, targetURL)

	client := &http.Client{
		Timeout: 30 * time.Second,
	}

	req, err := http.NewRequest("GET", fullURL, nil)
	if err != nil {
		return nil, err
	}

	return client.Do(req)
}
