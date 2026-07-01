package http

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	"coffee-pos/backend/internal/adapters/security"
	appauth "coffee-pos/backend/internal/app/auth"
	appmenu "coffee-pos/backend/internal/app/menu"
	domainmenu "coffee-pos/backend/internal/domain/menu"
)

func TestCashierMenuRequiresAuthentication(t *testing.T) {
	fixture := newMenuRouterFixture(t, appmenu.CashierMenu{})
	request := httptest.NewRequest(http.MethodGet, "/api/pos/menu", nil)
	response := httptest.NewRecorder()

	fixture.router.ServeHTTP(response, request)

	assertJSONError(t, response, http.StatusUnauthorized, "unauthorized")
}

func TestCashierMenuReturnsAuthenticatedReadModel(t *testing.T) {
	fixture := newMenuRouterFixture(t, appmenu.CashierMenu{
		Categories: []appmenu.CashierMenuCategory{{
			Name: "Coffee",
			Slug: "coffee",
			Items: []appmenu.CashierMenuItem{{
				Name:           "Kopi Susu",
				Slug:           "kopi-susu",
				PriceRp:        18000,
				ImagePath:      "/menu/kopi-susu.png",
				PopularityRank: 7,
				BestSeller:     true,
				Promo:          true,
				Iced:           true,
				LowSugar:       true,
				NewArrival:     true,
				ModifierGroups: []appmenu.CashierModifierGroup{{
					Name:          "Temperature",
					Slug:          "temperature",
					Required:      true,
					SelectionType: "single",
					Options: []appmenu.CashierModifierOption{{
						Name:         "Hot",
						Slug:         "hot",
						PriceDeltaRp: 0,
					}},
				}},
			}},
		}},
	})
	loginResponse := fixture.login(t)
	cookie := assertSingleCookie(t, loginResponse, fixture.cookieName)
	request := httptest.NewRequest(http.MethodGet, "/api/pos/menu", nil)
	request.AddCookie(cookie)
	response := httptest.NewRecorder()

	fixture.router.ServeHTTP(response, request)

	if response.Code != http.StatusOK {
		t.Fatalf("status = %d, want 200", response.Code)
	}
	var body struct {
		Categories []struct {
			Slug  string `json:"slug"`
			Items []struct {
				Slug           string `json:"slug"`
				PriceRp        int64  `json:"priceRp"`
				ImagePath      string `json:"imagePath"`
				PopularityRank int    `json:"popularityRank"`
				BestSeller     bool   `json:"bestSeller"`
				Promo          bool   `json:"promo"`
				Iced           bool   `json:"iced"`
				LowSugar       bool   `json:"lowSugar"`
				NewArrival     bool   `json:"newArrival"`
				ModifierGroups []struct {
					Slug    string `json:"slug"`
					Options []struct {
						Slug string `json:"slug"`
					} `json:"options"`
				} `json:"modifierGroups"`
			} `json:"items"`
		} `json:"categories"`
	}
	if err := json.NewDecoder(response.Body).Decode(&body); err != nil {
		t.Fatalf("decode response: %v", err)
	}
	if got := body.Categories[0].Items[0].ModifierGroups[0].Options[0].Slug; got != "hot" {
		t.Fatalf("option slug = %q, want hot", got)
	}
	if got := body.Categories[0].Items[0].PriceRp; got != 18000 {
		t.Fatalf("priceRp = %d, want 18000", got)
	}
	item := body.Categories[0].Items[0]
	if item.ImagePath != "/menu/kopi-susu.png" ||
		item.PopularityRank != 7 ||
		!item.BestSeller ||
		!item.Promo ||
		!item.Iced ||
		!item.LowSugar ||
		!item.NewArrival {
		t.Fatalf("unexpected display metadata: %+v", item)
	}
}

type menuRouterFixture struct {
	router     http.Handler
	cookieName string
}

func newMenuRouterFixture(t *testing.T, cashierMenu appmenu.CashierMenu) menuRouterFixture {
	t.Helper()

	jakarta := time.FixedZone("Asia/Jakarta", 7*60*60)
	hasher := security.BcryptPINHash{}
	hash, err := hasher.HashPIN("123456")
	if err != nil {
		t.Fatalf("hash pin: %v", err)
	}
	authService := appauth.NewService(appauth.Dependencies{
		CashierPINHash: hash,
		Verifier:       hasher,
		Sessions:       security.NewInMemorySessionStore(),
		RateLimiter:    security.NewInMemoryRateLimiter(),
		SessionIDs:     &sequentialSessionIDGenerator{},
		Clock:          &mutableClock{now: time.Date(2026, 6, 30, 10, 0, 0, 0, jakarta)},
		Location:       jakarta,
	})
	menuService := appmenu.NewService(appmenu.Dependencies{Repository: &fakeHTTPMenuRepository{menu: cashierMenu}})
	cookieName := "coffee_pos_session"

	return menuRouterFixture{
		router: NewRouter(RouterOptions{
			AuthService: authService,
			MenuService: &menuService,
			Cookie: CookieConfig{
				Name:     cookieName,
				Path:     "/",
				SameSite: http.SameSiteLaxMode,
			},
		}),
		cookieName: cookieName,
	}
}

func (fixture menuRouterFixture) login(t *testing.T) *httptest.ResponseRecorder {
	t.Helper()

	request := httptest.NewRequest(http.MethodPost, "/api/auth/login", strings.NewReader(`{"pin":"123456"}`))
	request.RemoteAddr = "203.0.113.10:1234"
	response := httptest.NewRecorder()
	fixture.router.ServeHTTP(response, request)
	if response.Code != http.StatusOK {
		t.Fatalf("login status = %d, want 200", response.Code)
	}
	return response
}

type fakeHTTPMenuRepository struct {
	menu appmenu.CashierMenu
}

func (repo *fakeHTTPMenuRepository) SeedMenu(context.Context, domainmenu.Seed) error {
	return nil
}

func (repo *fakeHTTPMenuRepository) GetCashierMenu(context.Context) (appmenu.CashierMenu, error) {
	return repo.menu, nil
}
