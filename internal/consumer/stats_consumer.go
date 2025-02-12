package consumer

import (
	"context"
	"encoding/json"
	ckafka "github.com/confluentinc/confluent-kafka-go/v2/kafka"
	"github.com/vncats/otel-demo/internal/store"
	"math"
)

type StatsConsumer struct {
	*Consumer
}

// NewStatsConsumer returns new instance
func NewStatsConsumer(st store.IStore) (*StatsConsumer, error) {
	consumer, err := NewConsumer(Options{
		Group:   "movie_stats_consumer_group",
		Topic:   "movie.rating-created",
		Brokers: "localhost:9092",
		Handler: handleStatsConsumer(st),
	})
	if err != nil {
		return nil, err
	}

	return &StatsConsumer{consumer}, nil
}

func handleStatsConsumer(st store.IStore) func(context.Context, *ckafka.Message) {
	return func(ctx context.Context, msg *ckafka.Message) {
		rating := store.Rating{}
		if err := json.Unmarshal(msg.Value, &rating); err != nil {
			return
		}

		ratings, err := st.GetRatingsByMovie(ctx, rating.MovieID)
		if err != nil {
			return
		}
		if len(ratings) == 0 {
			return
		}

		stats := &store.Stats{
			Histogram: map[int]int{},
		}
		scoreSum := 0
		for _, r := range ratings {
			scoreSum += r.Score
			stats.NumRating++
			if _, ok := stats.Histogram[r.Score]; !ok {
				stats.Histogram[r.Score] = 0
			}
			stats.Histogram[r.Score]++
		}
		stats.AvgScore = math.Round(float64(scoreSum)*100/float64(stats.NumRating)) / 100

		_ = st.UpdateStats(ctx, rating.MovieID, stats)
	}
}
