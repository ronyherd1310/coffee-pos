package menu

import (
	"context"
	"fmt"

	domainmenu "coffee-pos/backend/internal/domain/menu"
)

type Dependencies struct {
	Repository SeedRepository
}

type Service struct {
	repository SeedRepository
}

func NewService(deps Dependencies) Service {
	return Service{repository: deps.Repository}
}

func (service Service) SeedInitialMenu(ctx context.Context) error {
	return service.Seed(ctx, domainmenu.ApprovedSeed())
}

func (service Service) Seed(ctx context.Context, seed domainmenu.Seed) error {
	if err := domainmenu.ValidateSeed(seed); err != nil {
		return fmt.Errorf("invalid menu seed: %w", err)
	}
	if service.repository == nil {
		return fmt.Errorf("persist menu seed: repository is required")
	}
	if err := service.repository.SeedMenu(ctx, seed); err != nil {
		return fmt.Errorf("persist menu seed: %w", err)
	}
	return nil
}
