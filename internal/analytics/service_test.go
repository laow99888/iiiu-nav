package analytics

import (
	"context"
	"reflect"
	"testing"
	"time"
)

func TestServiceRecordsLocalDayAndRetentionCutoff(t *testing.T) {
	t.Parallel()
	location := time.FixedZone("test", 8*60*60)
	repository := &fakeRepository{}
	service, err := New(Config{
		CurrentTime: func() time.Time {
			return time.Date(2026, time.August, 19, 16, 30, 0, 0, time.UTC)
		},
		Location:   location,
		Repository: repository,
	})
	if err != nil {
		t.Fatalf("create service: %v", err)
	}
	if err := service.RecordPageView(context.Background()); err != nil {
		t.Fatalf("record page view: %v", err)
	}
	if repository.incrementDay != "2026-08-20" || repository.cutoff != "2025-08-21" {
		t.Fatalf("unexpected day=%s cutoff=%s", repository.incrementDay, repository.cutoff)
	}
}

func TestServiceReturnsZeroFilledChronologicalSeries(t *testing.T) {
	t.Parallel()
	repository := &fakeRepository{series: []DailyPageViews{
		{Day: "2026-08-17", Views: 3},
		{Day: "2026-08-19", Views: 5},
	}}
	service, err := New(Config{
		CurrentTime: func() time.Time {
			return time.Date(2026, time.August, 19, 12, 0, 0, 0, time.UTC)
		},
		Location:   time.UTC,
		Repository: repository,
	})
	if err != nil {
		t.Fatalf("create service: %v", err)
	}
	series, err := service.PageViewSeries(context.Background(), 3)
	if err != nil {
		t.Fatalf("read series: %v", err)
	}
	want := []DailyPageViews{
		{Day: "2026-08-17", Views: 3},
		{Day: "2026-08-18", Views: 0},
		{Day: "2026-08-19", Views: 5},
	}
	if !reflect.DeepEqual(series, want) {
		t.Fatalf("unexpected series: %+v", series)
	}
	if repository.since != "2026-08-17" {
		t.Fatalf("unexpected lower bound %s", repository.since)
	}
}

type fakeRepository struct {
	cutoff       string
	incrementDay string
	series       []DailyPageViews
	since        string
}

func (repository *fakeRepository) IncrementPageViews(_ context.Context, day, cutoff string) error {
	repository.incrementDay = day
	repository.cutoff = cutoff
	return nil
}

func (repository *fakeRepository) PageViews(_ context.Context, since string) ([]DailyPageViews, error) {
	repository.since = since
	return repository.series, nil
}
