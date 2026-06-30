package http

import (
	"context"
	"encoding/json"
	"net/http"

	appauth "coffee-pos/backend/internal/app/auth"
	appmenu "coffee-pos/backend/internal/app/menu"
	apporders "coffee-pos/backend/internal/app/orders"
)

type healthResponse struct {
	Status  string `json:"status"`
	Service string `json:"service"`
}

type CookieConfig struct {
	Name     string
	Path     string
	Secure   bool
	SameSite http.SameSite
}

type RouterOptions struct {
	AuthService  *appauth.Service
	MenuService  *appmenu.Service
	OrderService orderService
	Cookie       CookieConfig
}

type orderService interface {
	CreatePaidOrder(context.Context, apporders.CreatePaidOrderInput) (apporders.PaidOrderDetail, apporders.CreatePaidOrderResult, error)
	CancelPaidOrder(context.Context, apporders.CancelPaidOrderInput) (apporders.PaidOrderDetail, apporders.CancelPaidOrderResult, error)
}

func NewRouter(options ...RouterOptions) http.Handler {
	mux := http.NewServeMux()
	mux.HandleFunc("GET /api/health", handleHealth)

	if len(options) > 0 {
		cookieConfig := normalizeCookieConfig(options[0].Cookie)
		authHandlers := newAuthHandlers(options[0].AuthService, cookieConfig)
		menuHandlers := newMenuHandlers(options[0].MenuService)
		orderHandlers := newOrderHandlers(options[0].OrderService)
		authGuard := newAuthMiddleware(options[0].AuthService, cookieConfig.Name)
		mux.HandleFunc("POST /api/auth/login", authHandlers.handleLogin)
		mux.HandleFunc("POST /api/auth/logout", authHandlers.handleLogout)
		mux.HandleFunc("GET /api/auth/session", authHandlers.handleSession)
		mux.Handle("GET /api/pos/ping", authGuard.requireAuth(http.HandlerFunc(handleProtectedPOSPing)))
		mux.Handle("GET /api/pos/menu", authGuard.requireAuth(http.HandlerFunc(menuHandlers.handleCashierMenu)))
		mux.Handle("POST /api/pos/orders", authGuard.requireAuth(http.HandlerFunc(orderHandlers.handleCreatePaidOrder)))
		mux.Handle("POST /api/pos/orders/{orderId}/cancel", authGuard.requireAuth(http.HandlerFunc(orderHandlers.handleCancelPaidOrder)))
	}

	return mux
}

func handleHealth(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusOK)

	_ = json.NewEncoder(w).Encode(healthResponse{
		Status:  "ok",
		Service: "coffee-pos-backend",
	})
}

func handleProtectedPOSPing(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusOK)

	_ = json.NewEncoder(w).Encode(map[string]string{
		"status": "ok",
	})
}
