package message

import (
	"encoding/json"
	"math"
	"time"

	ckafka "github.com/confluentinc/confluent-kafka-go/v2/kafka"
	"github.com/vncats/otel-demo/internal/store"
	"github.com/vncats/otel-demo/pkg/kafka"
	"github.com/vncats/otel-demo/pkg/retry"
)

type StatsConsumer struct {
	*kafka.Consumer
}

// NewStatsConsumer returns new instance
func NewStatsConsumer(st store.IStore) (*StatsConsumer, error) {
	handler := statsHandler{store: st}
	consumer, err := kafka.NewConsumer(kafka.ConsumerOptions{
		Brokers:       "localhost:9092",
		Group:         "movie_stats_consumer_group",
		Topics:        []string{"movie.rating-created"},
		Offset:        kafka.OffsetEarliest,
		EnableTracing: true,
		MessageHandler: kafka.HandleWithRetry(handler.handleMessage, retry.Config{
			InitialInterval: 5 * time.Second,
			MaxInterval:     30 * time.Second,
			Multiplier:      2,
			MaxRetries:      5,
		}),
	})
	if err != nil {
		return nil, err
	}

	return &StatsConsumer{consumer}, nil
}

type statsHandler struct {
	store store.IStore
}

func (s *statsHandler) handleMessage(msg *ckafka.Message) error {
	ctx, span := startSpan(msg, "handle message")
	defer span.End()

	rating := store.Rating{}
	if err := json.Unmarshal(msg.Value, &rating); err != nil {
		return err
	}

	counts, err := s.store.GetRatingCounts(ctx, rating.MovieID)
	if err != nil {
		return err
	}
	if len(counts) == 0 {
		return nil
	}

	stats := &store.Stats{
		Histogram: map[int]int{},
	}

	scoreSum := 0
	for _, group := range counts {
		stats.NumRating += group.Count
		stats.Histogram[group.Score] = group.Count
		scoreSum += group.Score * group.Count
	}
	stats.AvgScore = math.Round(float64(scoreSum)*100/float64(stats.NumRating)) / 100

	return s.store.UpdateStats(ctx, rating.MovieID, stats)
}
