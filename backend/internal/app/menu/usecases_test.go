package menu

import (
	"context"
	"errors"
	"strings"
	"testing"

	domainmenu "coffee-pos/backend/internal/domain/menu"
)

func TestServiceSeedInitialMenuPersistsApprovedSeed(t *testing.T) {
	repository := &fakeSeedRepository{}
	service := NewService(Dependencies{Repository: repository})

	if err := service.SeedInitialMenu(context.Background()); err != nil {
		t.Fatalf("expected seed to succeed: %v", err)
	}

	if repository.calls != 1 {
		t.Fatalf("expected repository to be called once, got %d", repository.calls)
	}
	if repository.seed.Category.Name != "Coffee" {
		t.Fatalf("expected approved seed to be persisted, got %+v", repository.seed)
	}
}

func TestServiceSeedRejectsInvalidSeedBeforeRepositoryWrite(t *testing.T) {
	repository := &fakeSeedRepository{}
	service := NewService(Dependencies{Repository: repository})
	seed := domainmenu.ApprovedSeed()
	seed.Items[0].PriceRp = -1

	err := service.Seed(context.Background(), seed)
	if err == nil {
		t.Fatal("expected validation failure")
	}
	if !strings.Contains(err.Error(), "invalid menu seed") {
		t.Fatalf("expected invalid seed error, got %q", err.Error())
	}
	if repository.calls != 0 {
		t.Fatalf("expected repository not to be called, got %d calls", repository.calls)
	}
}

func TestServiceSeedReturnsRepositoryFailures(t *testing.T) {
	repository := &fakeSeedRepository{err: errors.New("write failed")}
	service := NewService(Dependencies{Repository: repository})

	err := service.SeedInitialMenu(context.Background())
	if err == nil {
		t.Fatal("expected repository failure")
	}
	if !strings.Contains(err.Error(), "persist menu seed") {
		t.Fatalf("expected persistence error, got %q", err.Error())
	}
}

type fakeSeedRepository struct {
	calls int
	seed  domainmenu.Seed
	err   error
}

func (repo *fakeSeedRepository) SeedMenu(_ context.Context, seed domainmenu.Seed) error {
	repo.calls++
	repo.seed = seed
	return repo.err
}
