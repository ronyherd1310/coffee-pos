//go:build integration

package postgres

import (
	"context"
	"testing"

	domainmenu "coffee-pos/backend/internal/domain/menu"

	"coffee-pos/backend/internal/adapters/postgres/sqlc"
)

func TestMenuRepositorySeedsApprovedMenuIdempotently(t *testing.T) {
	ctx := context.Background()
	db := startPostgresTestDB(t)
	applyTestMigrations(t, db)

	repository := NewMenuRepository(db)
	seed := domainmenu.ApprovedSeed()

	if err := repository.SeedMenu(ctx, seed); err != nil {
		t.Fatalf("expected first seed to succeed: %v", err)
	}
	if err := repository.SeedMenu(ctx, seed); err != nil {
		t.Fatalf("expected second seed to succeed: %v", err)
	}

	queries := sqlc.New(db)
	categories, err := queries.ListMenuCategories(ctx)
	if err != nil {
		t.Fatalf("list categories: %v", err)
	}
	if len(categories) != 1 || categories[0].Name != "Coffee" || categories[0].Slug != "coffee" {
		t.Fatalf("expected one Coffee category, got %+v", categories)
	}

	items, err := queries.ListMenuItems(ctx)
	if err != nil {
		t.Fatalf("list menu items: %v", err)
	}
	assertItemPrices(t, items, map[string]int32{
		"Americano": 18000,
		"Latte":     25000,
	})

	groups, err := queries.ListModifierGroups(ctx)
	if err != nil {
		t.Fatalf("list modifier groups: %v", err)
	}
	groupIDs := assertGroups(t, groups, []string{"Temperature", "Sugar"})

	options, err := queries.ListModifierOptions(ctx)
	if err != nil {
		t.Fatalf("list modifier options: %v", err)
	}
	assertOptions(t, options, groupIDs, map[string][]string{
		"Temperature": {"Hot", "Iced"},
		"Sugar":       {"Normal", "Less sugar", "No sugar"},
	})

	linkCount, err := queries.CountMenuItemModifierGroups(ctx)
	if err != nil {
		t.Fatalf("count item modifier links: %v", err)
	}
	if linkCount != 4 {
		t.Fatalf("expected 4 item/group links, got %d", linkCount)
	}
}

func assertItemPrices(t *testing.T, items []sqlc.ListMenuItemsRow, expected map[string]int32) {
	t.Helper()

	if len(items) != len(expected) {
		t.Fatalf("expected %d items, got %+v", len(expected), items)
	}
	for _, item := range items {
		price, exists := expected[item.Name]
		if !exists {
			t.Fatalf("unexpected item %+v", item)
		}
		if item.PriceRp != price || !item.Active {
			t.Fatalf("unexpected item values %+v", item)
		}
	}
}

func assertGroups(t *testing.T, groups []sqlc.ListModifierGroupsRow, expectedNames []string) map[int64]string {
	t.Helper()

	if len(groups) != len(expectedNames) {
		t.Fatalf("expected %d groups, got %+v", len(expectedNames), groups)
	}
	groupIDs := map[int64]string{}
	for index, name := range expectedNames {
		group := groups[index]
		if group.Name != name {
			t.Fatalf("expected group %d to be %q, got %+v", index, name, group)
		}
		if !group.Required || group.SelectionType != string(domainmenu.SelectionSingle) {
			t.Fatalf("unexpected group values %+v", group)
		}
		groupIDs[group.ID] = group.Name
	}
	return groupIDs
}

func assertOptions(t *testing.T, options []sqlc.ListModifierOptionsRow, groupIDs map[int64]string, expected map[string][]string) {
	t.Helper()

	if len(options) != 5 {
		t.Fatalf("expected 5 options, got %+v", options)
	}

	seen := map[string][]string{}
	for _, option := range options {
		groupName := groupIDs[option.ModifierGroupID]
		if groupName == "" {
			t.Fatalf("option references unexpected group id: %+v", option)
		}
		if option.PriceDeltaRp != 0 {
			t.Fatalf("expected zero price delta, got %+v", option)
		}
		seen[groupName] = append(seen[groupName], option.Name)
	}

	for groupName, expectedOptions := range expected {
		actualOptions := seen[groupName]
		if len(actualOptions) != len(expectedOptions) {
			t.Fatalf("expected options %v for %q, got %v", expectedOptions, groupName, actualOptions)
		}
		for index, expectedOption := range expectedOptions {
			if actualOptions[index] != expectedOption {
				t.Fatalf("expected option %d for %q to be %q, got %q", index, groupName, expectedOption, actualOptions[index])
			}
		}
	}
}
