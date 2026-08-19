package analytics

import (
	"context"
	"errors"
	"fmt"
	"time"
)

const (
	DefaultRetentionDays = 365
	MaximumSeriesDays    = 30
)

type DailyPageViews struct {
	Day   string
	Views int
}

type Repository interface {
	IncrementPageViews(ctx context.Context, day, retentionCutoff string) error
	PageViews(ctx context.Context, since string) ([]DailyPageViews, error)
}

type Config struct {
	CurrentTime   func() time.Time
	Location      *time.Location
	Repository    Repository
	RetentionDays int
}

type Service struct {
	currentTime   func() time.Time
	location      *time.Location
	repository    Repository
	retentionDays int
}

func New(config Config) (*Service, error) {
	if config.Repository == nil {
		return nil, errors.New("analytics repository is required")
	}
	if config.Location == nil {
		return nil, errors.New("analytics location is required")
	}
	if config.CurrentTime == nil {
		config.CurrentTime = time.Now
	}
	if config.RetentionDays <= 0 {
		config.RetentionDays = DefaultRetentionDays
	}
	return &Service{
		currentTime:   config.CurrentTime,
		location:      config.Location,
		repository:    config.Repository,
		retentionDays: config.RetentionDays,
	}, nil
}

func (service *Service) RecordPageView(ctx context.Context) error {
	today := service.today()
	cutoff := today.AddDate(0, 0, -(service.retentionDays - 1))
	if err := service.repository.IncrementPageViews(ctx, formatDay(today), formatDay(cutoff)); err != nil {
		return fmt.Errorf("record page view: %w", err)
	}
	return nil
}

func (service *Service) PageViewSeries(ctx context.Context, days int) ([]DailyPageViews, error) {
	if days < 1 || days > MaximumSeriesDays {
		return nil, errors.New("analytics series period is invalid")
	}
	start := service.today().AddDate(0, 0, -(days - 1))
	recorded, err := service.repository.PageViews(ctx, formatDay(start))
	if err != nil {
		return nil, fmt.Errorf("read page views: %w", err)
	}
	byDay := make(map[string]int, len(recorded))
	for _, item := range recorded {
		byDay[item.Day] = item.Views
	}
	series := make([]DailyPageViews, 0, days)
	for offset := 0; offset < days; offset++ {
		day := formatDay(start.AddDate(0, 0, offset))
		series = append(series, DailyPageViews{Day: day, Views: byDay[day]})
	}
	return series, nil
}

func (service *Service) today() time.Time {
	now := service.currentTime().In(service.location)
	return time.Date(now.Year(), now.Month(), now.Day(), 0, 0, 0, 0, service.location)
}

func formatDay(value time.Time) string {
	return value.Format(time.DateOnly)
}
