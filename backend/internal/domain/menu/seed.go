package menu

import (
	"fmt"
	"strings"
)

func ApprovedSeed() Seed {
	drinkModifiers := []string{"temperature", "sugar"}

	return Seed{
		Categories: []Category{
			{Name: "Coffee", Slug: "coffee"},
			{Name: "Tea", Slug: "tea"},
			{Name: "Snacks", Slug: "snacks"},
			{Name: "Seasonal", Slug: "seasonal"},
		},
		Items: []Item{
			{Name: "Americano", Slug: "americano", CategorySlug: "coffee", PriceRp: 18000, ModifierGroupSlugs: drinkModifiers, Display: ItemDisplay{ImagePath: "/menu/americano.png", PopularityRank: 10, BestSeller: true, Iced: true, LowSugar: true}},
			{Name: "Latte", Slug: "latte", CategorySlug: "coffee", PriceRp: 25000, ModifierGroupSlugs: drinkModifiers, Display: ItemDisplay{ImagePath: "/menu/latte.png", PopularityRank: 20, BestSeller: true, Iced: true, LowSugar: true}},
			{Name: "Cappuccino", Slug: "cappuccino", CategorySlug: "coffee", PriceRp: 25000, ModifierGroupSlugs: drinkModifiers, Display: ItemDisplay{ImagePath: "/menu/cappuccino.png", PopularityRank: 40, Iced: true, LowSugar: true}},
			{Name: "Mocha", Slug: "mocha", CategorySlug: "coffee", PriceRp: 28000, ModifierGroupSlugs: drinkModifiers, Display: ItemDisplay{ImagePath: "/menu/mocha.png", PopularityRank: 50, Promo: true, Iced: true, LowSugar: true}},
			{Name: "Flat White", Slug: "flat-white", CategorySlug: "coffee", PriceRp: 24000, ModifierGroupSlugs: drinkModifiers, Display: ItemDisplay{ImagePath: "/menu/flat-white.png", PopularityRank: 60, LowSugar: true}},
			{Name: "Espresso", Slug: "espresso", CategorySlug: "coffee", PriceRp: 15000, ModifierGroupSlugs: drinkModifiers, Display: ItemDisplay{ImagePath: "/menu/espresso.png", PopularityRank: 70, LowSugar: true}},
			{Name: "Iced Tea", Slug: "iced-tea", CategorySlug: "tea", PriceRp: 15000, ModifierGroupSlugs: drinkModifiers, Display: ItemDisplay{ImagePath: "/menu/iced-tea.png", PopularityRank: 80, Iced: true}},
			{Name: "Croissant", Slug: "croissant", CategorySlug: "snacks", PriceRp: 20000, Display: ItemDisplay{ImagePath: "/menu/croissant.png", PopularityRank: 90}},
			{Name: "Muffin", Slug: "muffin", CategorySlug: "snacks", PriceRp: 20000, Display: ItemDisplay{ImagePath: "/menu/muffin.png", PopularityRank: 100, NewArrival: true}},
			{Name: "Matcha Latte", Slug: "matcha-latte", CategorySlug: "seasonal", PriceRp: 28000, ModifierGroupSlugs: drinkModifiers, Display: ItemDisplay{ImagePath: "/menu/matcha-latte.png", PopularityRank: 30, Iced: true, LowSugar: true, NewArrival: true}},
			{Name: "Caramel Latte", Slug: "caramel-latte", CategorySlug: "seasonal", PriceRp: 28000, ModifierGroupSlugs: drinkModifiers, Display: ItemDisplay{ImagePath: "/menu/caramel-latte.png", PopularityRank: 110, Promo: true, Iced: true}},
			{Name: "Chocolate", Slug: "chocolate", CategorySlug: "seasonal", PriceRp: 25000, ModifierGroupSlugs: drinkModifiers, Display: ItemDisplay{ImagePath: "/menu/chocolate.png", PopularityRank: 120, NewArrival: true}},
		},
		ModifierGroups: []ModifierGroup{
			{
				Name:          "Temperature",
				Slug:          "temperature",
				Required:      true,
				SelectionType: SelectionSingle,
				Options: []ModifierOption{
					{Name: "Hot", Slug: "hot", PriceDeltaRp: 0},
					{Name: "Iced", Slug: "iced", PriceDeltaRp: 0},
				},
			},
			{
				Name:          "Sugar",
				Slug:          "sugar",
				Required:      true,
				SelectionType: SelectionSingle,
				Options: []ModifierOption{
					{Name: "Normal", Slug: "normal", PriceDeltaRp: 0},
					{Name: "Less sugar", Slug: "less-sugar", PriceDeltaRp: 0},
					{Name: "No sugar", Slug: "no-sugar", PriceDeltaRp: 0},
				},
			},
		},
	}
}

func ValidateSeed(seed Seed) error {
	if len(seed.Categories) == 0 {
		return fmt.Errorf("validate menu seed: categories are required")
	}
	categorySlugs, err := validateCategories(seed.Categories)
	if err != nil {
		return err
	}
	modifierGroupSlugs, err := validateModifierGroups(seed.ModifierGroups)
	if err != nil {
		return err
	}
	if len(seed.Items) == 0 {
		return fmt.Errorf("validate menu seed: items are required")
	}
	if len(seed.ModifierGroups) == 0 {
		return fmt.Errorf("validate menu seed: modifier groups are required")
	}
	if err := validateItems(seed.Items, categorySlugs, modifierGroupSlugs); err != nil {
		return err
	}
	return nil
}

func validateCategories(categories []Category) (map[string]struct{}, error) {
	seenNames := map[string]struct{}{}
	seenSlugs := map[string]struct{}{}
	for _, category := range categories {
		if isBlank(category.Name) {
			return nil, fmt.Errorf("validate menu seed: category name is required")
		}
		if isBlank(category.Slug) {
			return nil, fmt.Errorf("validate menu seed: category slug is required")
		}
		nameKey := normalizedName(category.Name)
		if _, exists := seenNames[nameKey]; exists {
			return nil, fmt.Errorf("validate menu seed: duplicate category name %q", category.Name)
		}
		seenNames[nameKey] = struct{}{}
		slugKey := normalizedName(category.Slug)
		if _, exists := seenSlugs[slugKey]; exists {
			return nil, fmt.Errorf("validate menu seed: duplicate category slug %q", category.Slug)
		}
		seenSlugs[slugKey] = struct{}{}
	}
	return seenSlugs, nil
}

func validateItems(items []Item, categorySlugs map[string]struct{}, modifierGroupSlugs map[string]struct{}) error {
	seenNames := map[string]struct{}{}
	for _, item := range items {
		if isBlank(item.Name) {
			return fmt.Errorf("validate menu seed: item name is required")
		}
		if isBlank(item.Slug) {
			return fmt.Errorf("validate menu seed: item slug is required")
		}
		if item.PriceRp < 0 {
			return fmt.Errorf("validate menu seed: item %q price cannot be negative", item.Name)
		}
		if err := validateItemDisplay(item); err != nil {
			return err
		}
		if _, exists := categorySlugs[normalizedName(item.CategorySlug)]; !exists {
			return fmt.Errorf("validate menu seed: item %q references unknown category %q", item.Name, item.CategorySlug)
		}
		seenItemGroups := map[string]struct{}{}
		for _, groupSlug := range item.ModifierGroupSlugs {
			groupSlugKey := normalizedName(groupSlug)
			if _, exists := modifierGroupSlugs[groupSlugKey]; !exists {
				return fmt.Errorf("validate menu seed: item %q references unknown modifier group %q", item.Name, groupSlug)
			}
			if _, exists := seenItemGroups[groupSlugKey]; exists {
				return fmt.Errorf("validate menu seed: item %q references duplicate modifier group %q", item.Name, groupSlug)
			}
			seenItemGroups[groupSlugKey] = struct{}{}
		}
		nameKey := normalizedName(item.Name)
		if _, exists := seenNames[nameKey]; exists {
			return fmt.Errorf("validate menu seed: duplicate item name %q", item.Name)
		}
		seenNames[nameKey] = struct{}{}
	}
	return nil
}

func validateItemDisplay(item Item) error {
	if item.Display.ImagePath != "" && !strings.HasPrefix(item.Display.ImagePath, "/") {
		return fmt.Errorf("validate menu seed: item %q image path must be absolute", item.Name)
	}
	if item.Display.PopularityRank < 0 {
		return fmt.Errorf("validate menu seed: item %q popularity rank cannot be negative", item.Name)
	}
	return nil
}

func validateModifierGroups(groups []ModifierGroup) (map[string]struct{}, error) {
	seenNames := map[string]struct{}{}
	seenSlugs := map[string]struct{}{}
	for _, group := range groups {
		if isBlank(group.Name) {
			return nil, fmt.Errorf("validate menu seed: modifier group name is required")
		}
		if isBlank(group.Slug) {
			return nil, fmt.Errorf("validate menu seed: modifier group slug is required")
		}
		if group.SelectionType != SelectionSingle {
			return nil, fmt.Errorf("validate menu seed: modifier group %q must be single select", group.Name)
		}
		if group.Required && len(group.Options) == 0 {
			return nil, fmt.Errorf("validate menu seed: required modifier group %q needs options", group.Name)
		}
		nameKey := normalizedName(group.Name)
		if _, exists := seenNames[nameKey]; exists {
			return nil, fmt.Errorf("validate menu seed: duplicate modifier group name %q", group.Name)
		}
		seenNames[nameKey] = struct{}{}
		slugKey := normalizedName(group.Slug)
		if _, exists := seenSlugs[slugKey]; exists {
			return nil, fmt.Errorf("validate menu seed: duplicate modifier group slug %q", group.Slug)
		}
		seenSlugs[slugKey] = struct{}{}

		if err := validateModifierOptions(group); err != nil {
			return nil, err
		}
	}
	return seenSlugs, nil
}

func validateModifierOptions(group ModifierGroup) error {
	seenNames := map[string]struct{}{}
	for _, option := range group.Options {
		if isBlank(option.Name) {
			return fmt.Errorf("validate menu seed: option name is required for group %q", group.Name)
		}
		if isBlank(option.Slug) {
			return fmt.Errorf("validate menu seed: option slug is required for group %q", group.Name)
		}
		if option.PriceDeltaRp < 0 {
			return fmt.Errorf("validate menu seed: option %q price delta cannot be negative", option.Name)
		}
		nameKey := normalizedName(option.Name)
		if _, exists := seenNames[nameKey]; exists {
			return fmt.Errorf("validate menu seed: duplicate option name %q in group %q", option.Name, group.Name)
		}
		seenNames[nameKey] = struct{}{}
	}
	return nil
}

func isBlank(value string) bool {
	return strings.TrimSpace(value) == ""
}

func normalizedName(value string) string {
	return strings.ToLower(strings.TrimSpace(value))
}
