package cache

import (
	"context"
	"encoding/json"
	"errors"
	"github.com/redis/go-redis/extra/redisotel/v9"
	"github.com/redis/go-redis/v9"
	"github.com/vncats/otel-demo/internal/store"
	"time"
)

var ErrCacheMiss = errors.New("cache miss")

type ICache interface {
	GetMovies(ctx context.Context) ([]*store.Movie, error)
	Close() error
}

var _ ICache = (*Cache)(nil)

func NewCache(redisURI string, st store.IStore) (*Cache, error) {
	opt, err := redis.ParseURL(redisURI)
	if err != nil {
		return nil, err
	}

	rdb := redis.NewClient(opt)
	if err := redisotel.InstrumentTracing(rdb); err != nil {
		return nil, err
	}
	if err := redisotel.InstrumentMetrics(rdb); err != nil {
		return nil, err
	}

	return &Cache{store: st, client: rdb}, nil
}

type Cache struct {
	store  store.IStore
	client *redis.Client
}

func (c *Cache) GetMovies(ctx context.Context) ([]*store.Movie, error) {
	var movies []*store.Movie
	err := c.get(ctx, "movies", &movies)

	if err == nil {
		return movies, nil
	}
	if !errors.Is(err, ErrCacheMiss) {
		return nil, err
	}

	movies, err = c.store.GetMovies(ctx)
	if err != nil {
		return nil, err
	}
	if len(movies) > 0 {
		err = c.set(ctx, "movies", movies, 5*time.Second)
		if err != nil {
			return nil, err
		}
	}

	return movies, nil
}

func (c *Cache) Close() error {
	return c.client.Close()
}

func (c *Cache) get(ctx context.Context, key string, out interface{}) error {
	v, err := c.client.Get(ctx, key).Result()
	if errors.Is(err, redis.Nil) {
		return ErrCacheMiss
	}
	if err != nil {
		return err
	}

	return json.Unmarshal([]byte(v), &out)
}

func (c *Cache) set(ctx context.Context, key string, value interface{}, exp time.Duration) error {
	b, err := json.Marshal(value)
	if err != nil {
		return err
	}

	return c.client.Set(ctx, key, string(b), exp).Err()
}
