package http

import (
	"net/http"

	appmenu "coffee-pos/backend/internal/app/menu"
)

type menuHandlers struct {
	service *appmenu.Service
}

func newMenuHandlers(service *appmenu.Service) menuHandlers {
	return menuHandlers{service: service}
}

func (handler menuHandlers) handleCashierMenu(w http.ResponseWriter, r *http.Request) {
	if handler.service == nil {
		writeJSONError(w, http.StatusInternalServerError, "internal_error")
		return
	}

	menu, err := handler.service.GetCashierMenu(r.Context())
	if err != nil {
		writeJSONError(w, http.StatusInternalServerError, "internal_error")
		return
	}

	writeJSON(w, http.StatusOK, cashierMenuResponseFromApp(menu))
}

type cashierMenuResponse struct {
	Categories []cashierMenuCategoryResponse `json:"categories"`
}

type cashierMenuCategoryResponse struct {
	Name  string                    `json:"name"`
	Slug  string                    `json:"slug"`
	Items []cashierMenuItemResponse `json:"items"`
}

type cashierMenuItemResponse struct {
	Name           string                         `json:"name"`
	Slug           string                         `json:"slug"`
	PriceRp        int64                          `json:"priceRp"`
	ImagePath      string                         `json:"imagePath"`
	PopularityRank int64                          `json:"popularityRank"`
	BestSeller     bool                           `json:"bestSeller"`
	Promo          bool                           `json:"promo"`
	Iced           bool                           `json:"iced"`
	LowSugar       bool                           `json:"lowSugar"`
	NewArrival     bool                           `json:"newArrival"`
	ModifierGroups []cashierModifierGroupResponse `json:"modifierGroups"`
}

type cashierModifierGroupResponse struct {
	Name          string                          `json:"name"`
	Slug          string                          `json:"slug"`
	Required      bool                            `json:"required"`
	SelectionType string                          `json:"selectionType"`
	Options       []cashierModifierOptionResponse `json:"options"`
}

type cashierModifierOptionResponse struct {
	Name         string `json:"name"`
	Slug         string `json:"slug"`
	PriceDeltaRp int64  `json:"priceDeltaRp"`
}

func cashierMenuResponseFromApp(menu appmenu.CashierMenu) cashierMenuResponse {
	response := cashierMenuResponse{Categories: make([]cashierMenuCategoryResponse, 0, len(menu.Categories))}
	for _, category := range menu.Categories {
		categoryResponse := cashierMenuCategoryResponse{
			Name:  category.Name,
			Slug:  category.Slug,
			Items: make([]cashierMenuItemResponse, 0, len(category.Items)),
		}
		for _, item := range category.Items {
			itemResponse := cashierMenuItemResponse{
				Name:           item.Name,
				Slug:           item.Slug,
				PriceRp:        item.PriceRp,
				ImagePath:      item.ImagePath,
				PopularityRank: item.PopularityRank,
				BestSeller:     item.BestSeller,
				Promo:          item.Promo,
				Iced:           item.Iced,
				LowSugar:       item.LowSugar,
				NewArrival:     item.NewArrival,
				ModifierGroups: make([]cashierModifierGroupResponse, 0, len(item.ModifierGroups)),
			}
			for _, group := range item.ModifierGroups {
				groupResponse := cashierModifierGroupResponse{
					Name:          group.Name,
					Slug:          group.Slug,
					Required:      group.Required,
					SelectionType: group.SelectionType,
					Options:       make([]cashierModifierOptionResponse, 0, len(group.Options)),
				}
				for _, option := range group.Options {
					groupResponse.Options = append(groupResponse.Options, cashierModifierOptionResponse{
						Name:         option.Name,
						Slug:         option.Slug,
						PriceDeltaRp: option.PriceDeltaRp,
					})
				}
				itemResponse.ModifierGroups = append(itemResponse.ModifierGroups, groupResponse)
			}
			categoryResponse.Items = append(categoryResponse.Items, itemResponse)
		}
		response.Categories = append(response.Categories, categoryResponse)
	}
	return response
}
