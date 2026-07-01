package menu

import (
	"strings"
	"testing"
)

func TestApprovedSeedDefinesMVPMenu(t *testing.T) {
	seed := ApprovedSeed()

	if err := ValidateSeed(seed); err != nil {
		t.Fatalf("expected approved seed to validate: %v", err)
	}

	expectedCategories := []Category{
		{Name: "Coffee", Slug: "coffee"},
		{Name: "Tea", Slug: "tea"},
		{Name: "Snacks", Slug: "snacks"},
		{Name: "Seasonal", Slug: "seasonal"},
	}
	if len(seed.Categories) != len(expectedCategories) {
		t.Fatalf("expected %d categories, got %d", len(expectedCategories), len(seed.Categories))
	}
	for index, expectedCategory := range expectedCategories {
		if seed.Categories[index] != expectedCategory {
			t.Fatalf("expected category %d to be %+v, got %+v", index, expectedCategory, seed.Categories[index])
		}
	}

	expectedItems := map[string]int{
		"americano":     18000,
		"latte":         25000,
		"cappuccino":    25000,
		"mocha":         28000,
		"matcha-latte":  28000,
		"flat-white":    24000,
		"caramel-latte": 28000,
		"espresso":      15000,
		"iced-tea":      15000,
		"chocolate":     25000,
		"croissant":     20000,
		"muffin":        20000,
	}
	if len(seed.Items) != len(expectedItems) {
		t.Fatalf("expected %d items, got %d", len(expectedItems), len(seed.Items))
	}
	for _, item := range seed.Items {
		if expectedItems[item.Slug] != item.PriceRp {
			t.Fatalf("unexpected item %+v", item)
		}
		if item.CategorySlug == "" {
			t.Fatalf("expected item %q to declare a category slug", item.Name)
		}
	}

	expectedMetadata := map[string]struct {
		imagePath      string
		popularityRank int
		bestSeller     bool
		iced           bool
		lowSugar       bool
		newArrival     bool
	}{
		"americano":    {imagePath: "/menu/americano.png", popularityRank: 10, bestSeller: true, iced: true, lowSugar: true},
		"latte":        {imagePath: "/menu/latte.png", popularityRank: 20, bestSeller: true, iced: true, lowSugar: true},
		"matcha-latte": {imagePath: "/menu/matcha-latte.png", popularityRank: 30, iced: true, lowSugar: true, newArrival: true},
		"croissant":    {imagePath: "/menu/croissant.png", popularityRank: 90},
	}
	for _, item := range seed.Items {
		if item.Display.ImagePath == "" {
			t.Fatalf("expected item %q to have an image path", item.Name)
		}
		expected, ok := expectedMetadata[item.Slug]
		if !ok {
			continue
		}
		if item.Display.ImagePath != expected.imagePath ||
			item.Display.PopularityRank != expected.popularityRank ||
			item.Display.BestSeller != expected.bestSeller ||
			item.Display.Iced != expected.iced ||
			item.Display.LowSugar != expected.lowSugar ||
			item.Display.NewArrival != expected.newArrival {
			t.Fatalf("unexpected display metadata for %q: %+v", item.Name, item.Display)
		}
	}

	expectedGroups := map[string][]string{
		"Temperature": {"Hot", "Iced"},
		"Sugar":       {"Normal", "Less sugar", "No sugar"},
	}
	if len(seed.ModifierGroups) != len(expectedGroups) {
		t.Fatalf("expected %d modifier groups, got %d", len(expectedGroups), len(seed.ModifierGroups))
	}
	for _, group := range seed.ModifierGroups {
		if !group.Required {
			t.Fatalf("expected group %q to be required", group.Name)
		}
		if group.SelectionType != SelectionSingle {
			t.Fatalf("expected group %q to be single select, got %q", group.Name, group.SelectionType)
		}
		expectedOptions := expectedGroups[group.Name]
		if len(group.Options) != len(expectedOptions) {
			t.Fatalf("expected %d options for %q, got %d", len(expectedOptions), group.Name, len(group.Options))
		}
		for index, option := range group.Options {
			if option.Name != expectedOptions[index] {
				t.Fatalf("expected option %d for %q to be %q, got %q", index, group.Name, expectedOptions[index], option.Name)
			}
			if option.PriceDeltaRp != 0 {
				t.Fatalf("expected option %q to have zero price delta, got %d", option.Name, option.PriceDeltaRp)
			}
		}
	}

	for _, item := range seed.Items {
		if item.CategorySlug == "snacks" && len(item.ModifierGroupSlugs) != 0 {
			t.Fatalf("expected snack item %q to have no required drink modifiers, got %v", item.Name, item.ModifierGroupSlugs)
		}
		if item.CategorySlug != "snacks" && len(item.ModifierGroupSlugs) != len(expectedGroups) {
			t.Fatalf("expected drink item %q to have drink modifiers, got %v", item.Name, item.ModifierGroupSlugs)
		}
	}
}

func TestValidateSeedRejectsInvalidDefinitions(t *testing.T) {
	tests := []struct {
		name    string
		mutate  func(seed *Seed)
		wantErr string
	}{
		{
			name: "empty category name",
			mutate: func(seed *Seed) {
				seed.Categories[0].Name = " "
			},
			wantErr: "category name",
		},
		{
			name: "negative item price",
			mutate: func(seed *Seed) {
				seed.Items[0].PriceRp = -1
			},
			wantErr: "price",
		},
		{
			name: "missing required options",
			mutate: func(seed *Seed) {
				seed.ModifierGroups[0].Options = nil
			},
			wantErr: "options",
		},
		{
			name: "duplicate item names",
			mutate: func(seed *Seed) {
				seed.Items[1].Name = seed.Items[0].Name
			},
			wantErr: "duplicate",
		},
		{
			name: "unknown item category",
			mutate: func(seed *Seed) {
				seed.Items[0].CategorySlug = "unknown"
			},
			wantErr: "category",
		},
		{
			name: "unknown item modifier group",
			mutate: func(seed *Seed) {
				seed.Items[0].ModifierGroupSlugs = []string{"unknown"}
			},
			wantErr: "modifier group",
		},
		{
			name: "relative image path",
			mutate: func(seed *Seed) {
				seed.Items[0].Display.ImagePath = "menu/americano.png"
			},
			wantErr: "image path",
		},
		{
			name: "negative popularity rank",
			mutate: func(seed *Seed) {
				seed.Items[0].Display.PopularityRank = -1
			},
			wantErr: "popularity rank",
		},
		{
			name: "duplicate option names",
			mutate: func(seed *Seed) {
				seed.ModifierGroups[1].Options[1].Name = seed.ModifierGroups[1].Options[0].Name
			},
			wantErr: "duplicate",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			seed := ApprovedSeed()
			tt.mutate(&seed)

			err := ValidateSeed(seed)
			if err == nil {
				t.Fatal("expected validation to fail")
			}
			if !strings.Contains(err.Error(), tt.wantErr) {
				t.Fatalf("expected error containing %q, got %q", tt.wantErr, err.Error())
			}
		})
	}
}
