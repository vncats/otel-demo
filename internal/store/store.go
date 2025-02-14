package store

import (
	"context"
	"database/sql/driver"
	"encoding/json"
	"errors"
	"fmt"

	"github.com/uptrace/opentelemetry-go-extra/otelgorm"
	"github.com/vncats/otel-demo/pkg/prim"
	"gorm.io/driver/mysql"
	"gorm.io/gorm"
	"gorm.io/gorm/clause"
	"gorm.io/gorm/logger"
)

type Movie struct {
	ID    int    `json:"id"`
	Title string `json:"title"`
	Stats Stats  `json:"stats"`
}

type Stats struct {
	AvgScore  float64     `json:"avg_score"`
	NumRating int         `json:"num_rating"`
	Histogram map[int]int `json:"histogram"`
}

func (m *Stats) Scan(val interface{}) error {
	switch v := val.(type) {
	case []byte:
		return json.Unmarshal(v, &m)
	case string:
		return json.Unmarshal([]byte(v), &m)
	default:
		return fmt.Errorf("unsupported type: %T", v)
	}
}

func (m Stats) Value() (driver.Value, error) {
	return json.Marshal(m)
}

type UserAction struct {
	ID      int      `json:"id"`
	Payload prim.Map `json:"payload"`
}

type Rating struct {
	ID      int    `json:"id"`
	MovieID int    `json:"movie_id"`
	UID     string `json:"uid"`
	Key     string `json:"-" gorm:"uniqueIndex"`
	Score   int    `json:"score"`
}

type RatingCount struct {
	Score int
	Count int
}

type IStore interface {
	CreateUserAction(ctx context.Context, act *UserAction) error
	CreateRating(ctx context.Context, rating *Rating) error
	GetRatingsByMovie(ctx context.Context, movieID int) ([]*Rating, error)
	GetRatingCounts(ctx context.Context, movieID int) ([]*RatingCount, error)
	UpdateStats(ctx context.Context, movieID int, stats *Stats) error
	GetMovies(ctx context.Context) ([]*Movie, error)
}

var _ IStore = (*Store)(nil)

func NewStore(dbURI string) (*Store, error) {
	db, err := gorm.Open(mysql.Open(dbURI), &gorm.Config{
		Logger: logger.Default.LogMode(logger.Silent),
	})
	if err != nil {
		return nil, err
	}

	if err = db.Use(otelgorm.NewPlugin(otelgorm.WithoutQueryVariables())); err != nil {
		return nil, err
	}

	return &Store{db: db}, nil
}

type Store struct {
	db *gorm.DB
}

func (s *Store) CreateUserAction(ctx context.Context, act *UserAction) error {
	return s.db.WithContext(ctx).Create(act).Error
}

func (s *Store) UpdateStats(ctx context.Context, movieID int, stats *Stats) error {
	movie := Movie{ID: movieID}
	err := s.db.WithContext(ctx).Model(&movie).Updates(map[string]interface{}{
		"stats": stats,
	}).Error
	if err != nil {
		return err
	}

	return nil
}

func (s *Store) CreateRating(ctx context.Context, rating *Rating) error {
	rating.Key = fmt.Sprintf("%s:%d", rating.UID, rating.MovieID)
	return s.db.WithContext(ctx).Clauses(clause.OnConflict{
		Columns:   []clause.Column{{Name: "key"}},
		DoUpdates: clause.Assignments(map[string]interface{}{"score": rating.Score}),
	}).Create(rating).Error
}

func (s *Store) GetRatingsByMovie(ctx context.Context, movieID int) ([]*Rating, error) {
	tx := s.db.WithContext(ctx).Where("movie_id = ?", movieID)

	var ratings []*Rating
	err := tx.Find(&ratings).Error
	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return ratings, nil
		}
		return nil, err
	}

	return ratings, nil
}

func (s *Store) GetRatingCounts(ctx context.Context, movieID int) ([]*RatingCount, error) {
	var results []*RatingCount
	err := s.db.WithContext(ctx).Model(&Rating{}).
		Where("movie_id = ?", movieID).
		Select("score, count(*) as count").
		Group("score").
		Scan(&results).Error
	if err != nil {
		return nil, err
	}

	return results, nil
}

func (s *Store) GetMovies(ctx context.Context) ([]*Movie, error) {
	var movies []*Movie
	err := s.db.WithContext(ctx).Find(&movies).Error
	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return movies, nil
		}
		return nil, err
	}

	return movies, nil
}

func (s *Store) Migrate() error {
	err := s.db.AutoMigrate(&Movie{}, &Rating{}, &UserAction{})
	if err != nil {
		return err
	}

	return s.seedMovies()
}

func (s *Store) seedMovies() error {
	movies := []*Movie{
		{
			ID:    1,
			Title: "The Shawshank Redemption",
		},
		{
			ID:    2,
			Title: "The Godfather",
		},
		{
			ID:    3,
			Title: "The Dark Knight",
		},
	}
	return s.db.Clauses(clause.OnConflict{DoNothing: true}).Create(movies).Error
}
