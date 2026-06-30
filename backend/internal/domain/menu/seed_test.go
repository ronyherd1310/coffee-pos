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
	if seed.Category.Name != "Coffee" {
		t.Fatalf("expected Coffee category, got %q", seed.Category.Name)
	}

	expectedItems := map[string]int{
		"Americano": 18000,
		"Latte":     25000,
	}
	if len(seed.Items) != len(expectedItems) {
		t.Fatalf("expected %d items, got %d", len(expectedItems), len(seed.Items))
	}
	for _, item := range seed.Items {
		if expectedItems[item.Name] != item.PriceRp {
			t.Fatalf("unexpected item %+v", item)
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
				seed.Category.Name = " "
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
