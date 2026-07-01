package postgres

import (
	"context"
	"database/sql"
	"fmt"

	"coffee-pos/backend/internal/adapters/postgres/sqlc"
	appmenu "coffee-pos/backend/internal/app/menu"
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

func (repo MenuRepository) GetCashierMenu(ctx context.Context) (appmenu.CashierMenu, error) {
	rows, err := repo.db.QueryContext(ctx, `
		select
			c.id,
			c.name,
			c.slug,
			i.id,
			i.name,
			i.slug,
			i.price_rp,
			i.image_path,
			i.popularity_rank,
			i.best_seller,
			i.promo,
			i.iced,
			i.low_sugar,
			i.new_arrival,
			g.id,
			g.name,
			g.slug,
			g.required,
			g.selection_type,
			o.id,
			o.name,
			o.slug,
			o.price_delta_rp
		from menu_categories c
		join menu_items i on i.category_id = c.id and i.active = true
		left join menu_item_modifier_groups mig on mig.menu_item_id = i.id
		left join modifier_groups g on g.id = mig.modifier_group_id
		left join modifier_options o on o.modifier_group_id = g.id
		order by c.sort_order, c.id, i.sort_order, i.id, mig.sort_order, g.sort_order, g.id, o.sort_order, o.id
	`)
	if err != nil {
		return appmenu.CashierMenu{}, fmt.Errorf("query cashier menu: %w", err)
	}
	defer rows.Close()

	var menu appmenu.CashierMenu
	categoryIndexes := map[int64]int{}
	itemIndexes := map[int64]int{}
	groupIndexes := map[int64]map[int64]int{}

	for rows.Next() {
		var row cashierMenuRow
		if err := rows.Scan(
			&row.categoryID,
			&row.categoryName,
			&row.categorySlug,
			&row.itemID,
			&row.itemName,
			&row.itemSlug,
			&row.priceRp,
			&row.imagePath,
			&row.popularityRank,
			&row.bestSeller,
			&row.promo,
			&row.iced,
			&row.lowSugar,
			&row.newArrival,
			&row.groupID,
			&row.groupName,
			&row.groupSlug,
			&row.required,
			&row.selectionType,
			&row.optionID,
			&row.optionName,
			&row.optionSlug,
			&row.priceDeltaRp,
		); err != nil {
			return appmenu.CashierMenu{}, fmt.Errorf("scan cashier menu: %w", err)
		}

		categoryIndex, ok := categoryIndexes[row.categoryID]
		if !ok {
			categoryIndex = len(menu.Categories)
			categoryIndexes[row.categoryID] = categoryIndex
			menu.Categories = append(menu.Categories, appmenu.CashierMenuCategory{
				ID:   row.categoryID,
				Name: row.categoryName,
				Slug: row.categorySlug,
			})
		}

		itemIndex, ok := itemIndexes[row.itemID]
		if !ok {
			itemIndex = len(menu.Categories[categoryIndex].Items)
			itemIndexes[row.itemID] = itemIndex
			menu.Categories[categoryIndex].Items = append(menu.Categories[categoryIndex].Items, appmenu.CashierMenuItem{
				ID:      row.itemID,
				Name:    row.itemName,
				Slug:    row.itemSlug,
				PriceRp: int64(row.priceRp),
				ImagePath: func() string {
					if row.imagePath.Valid {
						return row.imagePath.String
					}
					return ""
				}(),
				PopularityRank: func() int64 {
					if row.popularityRank.Valid {
						return int64(row.popularityRank.Int32)
					}
					return 0
				}(),
				BestSeller: row.bestSeller,
				Promo:      row.promo,
				Iced:       row.iced,
				LowSugar:   row.lowSugar,
				NewArrival: row.newArrival,
			})
		}

		if !row.groupID.Valid {
			continue
		}
		if groupIndexes[row.itemID] == nil {
			groupIndexes[row.itemID] = map[int64]int{}
		}
		groupIndex, ok := groupIndexes[row.itemID][row.groupID.Int64]
		if !ok {
			groupIndex = len(menu.Categories[categoryIndex].Items[itemIndex].ModifierGroups)
			groupIndexes[row.itemID][row.groupID.Int64] = groupIndex
			menu.Categories[categoryIndex].Items[itemIndex].ModifierGroups = append(menu.Categories[categoryIndex].Items[itemIndex].ModifierGroups, appmenu.CashierModifierGroup{
				ID:            row.groupID.Int64,
				Name:          row.groupName.String,
				Slug:          row.groupSlug.String,
				Required:      row.required.Bool,
				SelectionType: row.selectionType.String,
			})
		}

		if row.optionID.Valid {
			groups := menu.Categories[categoryIndex].Items[itemIndex].ModifierGroups
			groups[groupIndex].Options = append(groups[groupIndex].Options, appmenu.CashierModifierOption{
				ID:           row.optionID.Int64,
				Name:         row.optionName.String,
				Slug:         row.optionSlug.String,
				PriceDeltaRp: int64(row.priceDeltaRp.Int32),
			})
		}
	}
	if err := rows.Err(); err != nil {
		return appmenu.CashierMenu{}, fmt.Errorf("iterate cashier menu: %w", err)
	}

	return menu, nil
}

func (repo MenuRepository) seedMenu(ctx context.Context, queries *sqlc.Queries, seed domainmenu.Seed) error {
	categoryIDs := make(map[string]int64, len(seed.Categories))
	for index, category := range seed.Categories {
		categoryID, err := queries.UpsertMenuCategory(ctx, sqlc.UpsertMenuCategoryParams{
			Name:      category.Name,
			Slug:      category.Slug,
			SortOrder: int32(index),
		})
		if err != nil {
			return fmt.Errorf("upsert menu category %q: %w", category.Name, err)
		}
		categoryIDs[category.Slug] = categoryID
	}

	itemIDs := make(map[string]int64, len(seed.Items))
	for index, item := range seed.Items {
		categoryID, ok := categoryIDs[item.CategorySlug]
		if !ok {
			return fmt.Errorf("upsert menu item %q: unknown category %q", item.Name, item.CategorySlug)
		}
		itemID, err := queries.UpsertMenuItem(ctx, sqlc.UpsertMenuItemParams{
			CategoryID:     categoryID,
			Name:           item.Name,
			Slug:           item.Slug,
			PriceRp:        int32(item.PriceRp),
			Active:         true,
			SortOrder:      int32(index),
			ImagePath:      nullString(item.Display.ImagePath),
			PopularityRank: nullInt32(item.Display.PopularityRank),
			BestSeller:     item.Display.BestSeller,
			Promo:          item.Display.Promo,
			Iced:           item.Display.Iced,
			LowSugar:       item.Display.LowSugar,
			NewArrival:     item.Display.NewArrival,
		})
		if err != nil {
			return fmt.Errorf("upsert menu item %q: %w", item.Name, err)
		}
		itemIDs[item.Slug] = itemID
	}

	groupIDs := make(map[string]int64, len(seed.ModifierGroups))
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
		groupIDs[group.Slug] = groupID

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

	for _, item := range seed.Items {
		itemID, ok := itemIDs[item.Slug]
		if !ok {
			return fmt.Errorf("link menu item %q to modifier groups: missing item id", item.Name)
		}
		for groupIndex, groupSlug := range item.ModifierGroupSlugs {
			groupID, ok := groupIDs[groupSlug]
			if !ok {
				return fmt.Errorf("link menu item %q to modifier group %q: missing group id", item.Name, groupSlug)
			}
			if err := queries.UpsertMenuItemModifierGroup(ctx, sqlc.UpsertMenuItemModifierGroupParams{
				MenuItemID:      itemID,
				ModifierGroupID: groupID,
				SortOrder:       int32(groupIndex),
			}); err != nil {
				return fmt.Errorf("link menu item %q to modifier group %q: %w", item.Name, groupSlug, err)
			}
		}
	}

	return nil
}

func nullString(value string) sql.NullString {
	return sql.NullString{String: value, Valid: value != ""}
}

func nullInt32(value int) sql.NullInt32 {
	return sql.NullInt32{Int32: int32(value), Valid: value > 0}
}

type cashierMenuRow struct {
	categoryID     int64
	categoryName   string
	categorySlug   string
	itemID         int64
	itemName       string
	itemSlug       string
	priceRp        int32
	imagePath      sql.NullString
	popularityRank sql.NullInt32
	bestSeller     bool
	promo          bool
	iced           bool
	lowSugar       bool
	newArrival     bool
	groupID        sql.NullInt64
	groupName      sql.NullString
	groupSlug      sql.NullString
	required       sql.NullBool
	selectionType  sql.NullString
	optionID       sql.NullInt64
	optionName     sql.NullString
	optionSlug     sql.NullString
	priceDeltaRp   sql.NullInt32
}
