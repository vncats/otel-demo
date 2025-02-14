package main

import (
	"context"
	"fmt"
	"math/rand"
	"net/http"
	"os"
	"os/signal"
	"sync"
	"time"

	"github.com/google/uuid"
)

var userAgents = []string{
	"Chrome/135.1.0",
	"Safari/537.3.0",
	"Firefox/235.1.0",
	"Edge/135.1.0",
	"Opera/135.1.0",
}

func randomSession() (userID string, userAgent string) {
	userID = fmt.Sprintf("user_%d", time.Now().Unix())
	userAgent = userAgents[rand.Intn(len(userAgents))]
	return
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
	interval := 500 * time.Millisecond

	var wg sync.WaitGroup
	wg.Add(1)
	go func() {
		defer wg.Done()
		intervalCall(ctx, interval, func() {
			userID, userAgent := randomSession()
			client.Get(ctx, "http://localhost:8080/movies", map[string]string{
				"user-agent":   userAgent,
				"x-user-id":    userID,
				"x-request-id": uuid.NewString(),
			})
			fmt.Printf("== %s (%s): gets all movies\n", userID, userAgent)
		})
	}()

	wg.Add(1)
	go func() {
		defer wg.Done()
		intervalCall(ctx, interval, func() {
			url := fmt.Sprintf("http://localhost:8080/movies/%d/ratings/%d", 1+rand.Intn(3), 1+rand.Intn(5))
			userID, userAgent := randomSession()
			client.Get(ctx, url, map[string]string{
				"user-agent":   userAgent,
				"x-user-id":    userID,
				"x-request-id": uuid.NewString(),
			})
			fmt.Printf("== %s (%s): rates a movie\n", userID, userAgent)
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
