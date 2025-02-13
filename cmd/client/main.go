package main

import (
	"context"
	"fmt"
	"github.com/google/uuid"
	"math/rand"
	"net/http"
	"os"
	"os/signal"
	"sync"
	"time"
)

type Session struct {
	UserID    string
	UserAgent string
}

var sessions = []Session{
	{
		UserID:    "jesse_1201",
		UserAgent: "Chrome/135.1.0",
	},
	{
		UserID:    "jesse_1201",
		UserAgent: "Safari/537.3.0",
	},
	{
		UserID:    "alice_2705",
		UserAgent: "Chrome/125.1.0",
	},
	{
		UserID:    "peter_8802",
		UserAgent: "Safari/135.1.0",
	},
}

type Client struct {
	*http.Client
}

func NewClient() *Client {
	c := &http.Client{
		Timeout: 5 * time.Second,
		Transport: &http.Transport{
			MaxIdleConns:    10,
			IdleConnTimeout: 30 * time.Second,
		},
	}
	return &Client{c}
}

func (c *Client) Get(ctx context.Context, url string, headers map[string]string) {
	req, _ := http.NewRequestWithContext(ctx, "GET", url, nil)
	req.Header.Set("Content-Type", "application/json")
	for k, v := range headers {
		req.Header.Set(k, v)
	}
	resp, _ := c.Do(req)
	if resp != nil {
		_ = resp.Body.Close()
	}
}

func main() {
	ctx, cancel := signal.NotifyContext(context.Background(), os.Interrupt)
	defer cancel()

	client := NewClient()

	var wg sync.WaitGroup
	wg.Add(1)
	go func() {
		defer wg.Done()
		intervalCall(ctx, 2*time.Second, func() {
			ss := sessions[rand.Intn(len(sessions))]
			client.Get(ctx, "http://localhost:8080/movies", map[string]string{
				"user-agent":   ss.UserAgent,
				"x-user-id":    ss.UserID,
				"x-request-id": uuid.NewString(),
			})
			fmt.Printf("== %s (%s): gets all movies\n", ss.UserID, ss.UserAgent)
		})
	}()

	wg.Add(1)
	go func() {
		defer wg.Done()
		intervalCall(ctx, 2*time.Second, func() {
			ss := sessions[rand.Intn(len(sessions))]
			url := fmt.Sprintf("http://localhost:8080/movies/%d/ratings/%d", 1+rand.Intn(3), 1+rand.Intn(5))
			client.Get(ctx, url, map[string]string{
				"user-agent":   ss.UserAgent,
				"x-user-id":    ss.UserID,
				"x-request-id": uuid.NewString(),
			})
			fmt.Printf("== %s (%s): rates a movie\n", ss.UserID, ss.UserAgent)
		})
	}()

	fmt.Println("== Simulating API calls...")

	<-ctx.Done()

	cancel()
	wg.Wait()
}

func intervalCall(ctx context.Context, interval time.Duration, fn func()) {
	ticker := time.NewTicker(interval)
	defer ticker.Stop()
	for {
		select {
		case <-ticker.C:
			fn()
		case <-ctx.Done():
			return
		}
	}
}
