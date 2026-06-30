package postgres

import (
	"context"
	"database/sql"
	"fmt"

	"coffee-pos/backend/internal/adapters/postgres/sqlc"
	domainmenu "coffee-pos/backend/internal/domain/menu"
)

type MenuRepository struct {
	db *sql.DB
}

func NewMenuRepository(db *sql.DB) MenuRepository {
	return MenuRepository{db: db}
}

func (repo MenuRepository) SeedMenu(ctx context.Context, seed domainmenu.Seed) error {
	tx, err := repo.db.BeginTx(ctx, nil)
	if err != nil {
		return fmt.Errorf("begin menu seed transaction: %w", err)
	}

	queries := sqlc.New(repo.db).WithTx(tx)
	if err := repo.seedMenu(ctx, queries, seed); err != nil {
		_ = tx.Rollback()
		return err
	}
	if err := tx.Commit(); err != nil {
		return fmt.Errorf("commit menu seed transaction: %w", err)
	}
	return nil
}

func (repo MenuRepository) seedMenu(ctx context.Context, queries *sqlc.Queries, seed domainmenu.Seed) error {
	categoryID, err := queries.UpsertMenuCategory(ctx, sqlc.UpsertMenuCategoryParams{
		Name:      seed.Category.Name,
		Slug:      seed.Category.Slug,
		SortOrder: 0,
	})
	if err != nil {
		return fmt.Errorf("upsert menu category %q: %w", seed.Category.Name, err)
	}

	itemIDs := make([]int64, 0, len(seed.Items))
	for index, item := range seed.Items {
		itemID, err := queries.UpsertMenuItem(ctx, sqlc.UpsertMenuItemParams{
			CategoryID: categoryID,
			Name:       item.Name,
			Slug:       item.Slug,
			PriceRp:    int32(item.PriceRp),
			Active:     true,
			SortOrder:  int32(index),
		})
		if err != nil {
			return fmt.Errorf("upsert menu item %q: %w", item.Name, err)
		}
		itemIDs = append(itemIDs, itemID)
	}

	for groupIndex, group := range seed.ModifierGroups {
		groupID, err := queries.UpsertModifierGroup(ctx, sqlc.UpsertModifierGroupParams{
			Name:          group.Name,
			Slug:          group.Slug,
			Required:      group.Required,
			SelectionType: string(group.SelectionType),
			SortOrder:     int32(groupIndex),
		})
		if err != nil {
			return fmt.Errorf("upsert modifier group %q: %w", group.Name, err)
		}

		for _, itemID := range itemIDs {
			if err := queries.UpsertMenuItemModifierGroup(ctx, sqlc.UpsertMenuItemModifierGroupParams{
				MenuItemID:      itemID,
				ModifierGroupID: groupID,
				SortOrder:       int32(groupIndex),
			}); err != nil {
				return fmt.Errorf("link menu item to modifier group %q: %w", group.Name, err)
			}
		}

		for optionIndex, option := range group.Options {
			if _, err := queries.UpsertModifierOption(ctx, sqlc.UpsertModifierOptionParams{
				ModifierGroupID: groupID,
				Name:            option.Name,
				Slug:            option.Slug,
				PriceDeltaRp:    int32(option.PriceDeltaRp),
				SortOrder:       int32(optionIndex),
			}); err != nil {
				return fmt.Errorf("upsert modifier option %q: %w", option.Name, err)
			}
		}
	}

	return nil
}
