package menu

type SelectionType string

const SelectionSingle SelectionType = "single"

type Category struct {
	Name string
	Slug string
}

type Item struct {
	Name    string
	Slug    string
	PriceRp int
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
	Category       Category
	Items          []Item
	ModifierGroups []ModifierGroup
}
