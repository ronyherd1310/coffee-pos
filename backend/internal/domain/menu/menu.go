package menu

type SelectionType string

const SelectionSingle SelectionType = "single"

type Category struct {
	Name string
	Slug string
}

type Item struct {
	Name               string
	Slug               string
	CategorySlug       string
	PriceRp            int
	ModifierGroupSlugs []string
	Display            ItemDisplay
}

type ItemDisplay struct {
	ImagePath      string
	PopularityRank int
	BestSeller     bool
	Promo          bool
	Iced           bool
	LowSugar       bool
	NewArrival     bool
}

type ModifierGroup struct {
	Name          string
	Slug          string
	Required      bool
	SelectionType SelectionType
	Options       []ModifierOption
}

type ModifierOption struct {
	Name         string
	Slug         string
	PriceDeltaRp int
}

type Seed struct {
	Categories     []Category
	Items          []Item
	ModifierGroups []ModifierGroup
}
