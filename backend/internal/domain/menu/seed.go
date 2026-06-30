package menu

import (
	"fmt"
	"strings"
)

func ApprovedSeed() Seed {
	return Seed{
		Category: Category{Name: "Coffee", Slug: "coffee"},
		Items: []Item{
			{Name: "Americano", Slug: "americano", PriceRp: 18000},
			{Name: "Latte", Slug: "latte", PriceRp: 25000},
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
	if isBlank(seed.Category.Name) {
		return fmt.Errorf("validate menu seed: category name is required")
	}
	if isBlank(seed.Category.Slug) {
		return fmt.Errorf("validate menu seed: category slug is required")
	}
	if len(seed.Items) == 0 {
		return fmt.Errorf("validate menu seed: items are required")
	}
	if err := validateItems(seed.Items); err != nil {
		return err
	}
	if len(seed.ModifierGroups) == 0 {
		return fmt.Errorf("validate menu seed: modifier groups are required")
	}
	if err := validateModifierGroups(seed.ModifierGroups); err != nil {
		return err
	}
	return nil
}

func validateItems(items []Item) error {
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
		nameKey := normalizedName(item.Name)
		if _, exists := seenNames[nameKey]; exists {
			return fmt.Errorf("validate menu seed: duplicate item name %q", item.Name)
		}
		seenNames[nameKey] = struct{}{}
	}
	return nil
}

func validateModifierGroups(groups []ModifierGroup) error {
	seenNames := map[string]struct{}{}
	for _, group := range groups {
		if isBlank(group.Name) {
			return fmt.Errorf("validate menu seed: modifier group name is required")
		}
		if isBlank(group.Slug) {
			return fmt.Errorf("validate menu seed: modifier group slug is required")
		}
		if group.SelectionType != SelectionSingle {
			return fmt.Errorf("validate menu seed: modifier group %q must be single select", group.Name)
		}
		if group.Required && len(group.Options) == 0 {
			return fmt.Errorf("validate menu seed: required modifier group %q needs options", group.Name)
		}
		nameKey := normalizedName(group.Name)
		if _, exists := seenNames[nameKey]; exists {
			return fmt.Errorf("validate menu seed: duplicate modifier group name %q", group.Name)
		}
		seenNames[nameKey] = struct{}{}

		if err := validateModifierOptions(group); err != nil {
			return err
		}
	}
	return nil
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
