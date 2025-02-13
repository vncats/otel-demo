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
)

type Session struct {
	UserID    string
	SessionID string
	UserAgent string
}

var sessions = []Session{
	{
		UserID:    "jesse_1201",
		SessionID: "3f674707-3ef2-4276-b456-7293df489275",
		UserAgent: "Chrome/135.1.0",
	},
	{
		UserID:    "jesse_1201",
		SessionID: "ead40ffe-b168-4ad5-b911-2a3e3d2ee6f7",
		UserAgent: "Safari/537.3.0",
	},
	{
		UserID:    "alice_2705",
		SessionID: "10748401-471d-41f3-8df5-aadd89f7fcc0",
		UserAgent: "Chrome/125.1.0",
	},
	{
		UserID:    "peter_8802",
		SessionID: "66b41dcb-ec88-4653-a599-459025d26ce9",
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
			baggage := fmt.Sprintf("user_id=%s,session_id=%s,user_agent=%s", ss.UserID, ss.SessionID, ss.UserAgent)
			client.Get(ctx, "http://localhost:8080/movies", map[string]string{
				"baggage":    baggage,
				"user-agent": ss.UserAgent,
				"x-user-id":  ss.UserID,
			})
			fmt.Printf("== %s (%s): gets all movies\n", ss.UserID, ss.UserAgent)
		})
	}()

	wg.Add(1)
	go func() {
		defer wg.Done()
		intervalCall(ctx, 2*time.Second, func() {
			ss := sessions[rand.Intn(len(sessions))]
			baggage := fmt.Sprintf("user_id=%s,session_id=%s,user_agent=%s", ss.UserID, ss.SessionID, ss.UserAgent)
			url := fmt.Sprintf("http://localhost:8080/movies/%d/ratings/%d", 1+rand.Intn(3), 1+rand.Intn(5))
			client.Get(ctx, url, map[string]string{
				"baggage":    baggage,
				"user-agent": ss.UserAgent,
				"x-user-id":  ss.UserID,
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
