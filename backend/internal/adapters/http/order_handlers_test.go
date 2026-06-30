package http

import (
	"context"
	"errors"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	"coffee-pos/backend/internal/adapters/security"
	appauth "coffee-pos/backend/internal/app/auth"
	appmenu "coffee-pos/backend/internal/app/menu"
	apporders "coffee-pos/backend/internal/app/orders"
	domainmenu "coffee-pos/backend/internal/domain/menu"
)

func TestCreatePaidOrderRequiresAuthentication(t *testing.T) {
	fixture := newOrderRouterFixture(t, fakeOrderService{})
	request := httptest.NewRequest(http.MethodPost, "/api/pos/orders", strings.NewReader(`{}`))
	response := httptest.NewRecorder()

	fixture.router.ServeHTTP(response, request)

	assertJSONError(t, response, http.StatusUnauthorized, "unauthorized")
}

func TestCreatePaidOrderRejectsUnknownAndForbiddenFields(t *testing.T) {
	fixture := newOrderRouterFixture(t, fakeOrderService{})
	cookie := assertSingleCookie(t, fixture.login(t), fixture.cookieName)

	tests := []struct {
		name string
		body string
		code string
	}{
		{name: "unknown", body: `{"clientRequestId":"11111111-1111-4111-8111-111111111111","paymentMethod":"cash","lines":[],"extra":true}`, code: "unknown_field"},
		{name: "forbidden", body: `{"clientRequestId":"11111111-1111-4111-8111-111111111111","paymentMethod":"cash","totalRp":1,"lines":[]}`, code: "forbidden_field"},
	}
	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			request := httptest.NewRequest(http.MethodPost, "/api/pos/orders", strings.NewReader(test.body))
			request.AddCookie(cookie)
			response := httptest.NewRecorder()

			fixture.router.ServeHTTP(response, request)

			assertJSONError(t, response, http.StatusBadRequest, test.code)
		})
	}
}

func TestCreatePaidOrderRejectsMalformedPayloads(t *testing.T) {
	fixture := newOrderRouterFixture(t, fakeOrderService{})
	cookie := assertSingleCookie(t, fixture.login(t), fixture.cookieName)

	tests := []struct {
		name string
		body string
		code string
	}{
		{name: "malformed json", body: `{"clientRequestId":`, code: "invalid_json"},
		{name: "multiple json values", body: `{"clientRequestId":"11111111-1111-4111-8111-111111111111","paymentMethod":"cash","lines":[]} {}`, code: "invalid_json"},
		{name: "null client request id", body: `{"clientRequestId":null,"paymentMethod":"cash","lines":[]}`, code: "invalid_field_type"},
		{name: "wrong payment method type", body: `{"clientRequestId":"11111111-1111-4111-8111-111111111111","paymentMethod":1,"lines":[]}`, code: "invalid_field_type"},
		{name: "missing lines", body: `{"clientRequestId":"11111111-1111-4111-8111-111111111111","paymentMethod":"cash"}`, code: "missing_field"},
		{name: "null lines", body: `{"clientRequestId":"11111111-1111-4111-8111-111111111111","paymentMethod":"cash","lines":null}`, code: "invalid_field_type"},
		{name: "wrong line type", body: `{"clientRequestId":"11111111-1111-4111-8111-111111111111","paymentMethod":"cash","lines":[1]}`, code: "invalid_field_type"},
		{name: "wrong quantity type", body: `{"clientRequestId":"11111111-1111-4111-8111-111111111111","paymentMethod":"cash","lines":[{"menuItemSlug":"kopi-susu","quantity":"1","modifiers":[]}]}`, code: "invalid_field_type"},
		{name: "null modifiers", body: `{"clientRequestId":"11111111-1111-4111-8111-111111111111","paymentMethod":"cash","lines":[{"menuItemSlug":"kopi-susu","quantity":1,"modifiers":null}]}`, code: "invalid_field_type"},
		{name: "wrong modifier field type", body: `{"clientRequestId":"11111111-1111-4111-8111-111111111111","paymentMethod":"cash","lines":[{"menuItemSlug":"kopi-susu","quantity":1,"modifiers":[{"groupSlug":1,"optionSlug":"hot"}]}]}`, code: "invalid_field_type"},
	}
	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			request := httptest.NewRequest(http.MethodPost, "/api/pos/orders", strings.NewReader(test.body))
			request.AddCookie(cookie)
			response := httptest.NewRecorder()

			fixture.router.ServeHTTP(response, request)

			assertJSONError(t, response, http.StatusBadRequest, test.code)
		})
	}
}

func TestCreatePaidOrderReturnsCreatedDetail(t *testing.T) {
	service := fakeOrderService{detail: apporders.PaidOrderDetail{
		OrderID:       "1",
		QueueNumber:   1,
		BusinessDate:  "2026-06-30",
		Status:        "paid",
		PaymentMethod: "cash",
		PaidAt:        time.Date(2026, 6, 30, 10, 0, 0, 0, time.UTC),
		TotalRp:       18000,
	}}
	fixture := newOrderRouterFixture(t, service)
	cookie := assertSingleCookie(t, fixture.login(t), fixture.cookieName)
	request := httptest.NewRequest(http.MethodPost, "/api/pos/orders", strings.NewReader(`{
		"clientRequestId":"11111111-1111-4111-8111-111111111111",
		"paymentMethod":"cash",
		"lines":[{"menuItemSlug":"kopi-susu","quantity":1,"modifiers":[]}]
	}`))
	request.AddCookie(cookie)
	response := httptest.NewRecorder()

	fixture.router.ServeHTTP(response, request)

	if response.Code != http.StatusCreated {
		t.Fatalf("status = %d, want 201", response.Code)
	}
}

func TestCreatePaidOrderMapsServiceResultsAndErrors(t *testing.T) {
	tests := []struct {
		name       string
		service    fakeOrderService
		wantStatus int
		wantCode   string
	}{
		{name: "existing idempotency returns ok", service: fakeOrderService{detail: minimalPaidOrderDetail(), result: apporders.CreatePaidOrderExisting}, wantStatus: http.StatusOK},
		{name: "invalid client request id", service: fakeOrderService{err: apporders.ErrInvalidClientRequestID}, wantStatus: http.StatusBadRequest, wantCode: "invalid_client_request_id"},
		{name: "invalid order", service: fakeOrderService{err: apporders.ErrInvalidOrder}, wantStatus: http.StatusUnprocessableEntity, wantCode: "invalid_order"},
		{name: "idempotency conflict", service: fakeOrderService{err: apporders.ErrIdempotencyConflict}, wantStatus: http.StatusConflict, wantCode: "idempotency_conflict"},
		{name: "repository failure", service: fakeOrderService{err: errors.New("database unavailable")}, wantStatus: http.StatusInternalServerError, wantCode: "internal_error"},
	}
	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			fixture := newOrderRouterFixture(t, test.service)
			cookie := assertSingleCookie(t, fixture.login(t), fixture.cookieName)
			request := httptest.NewRequest(http.MethodPost, "/api/pos/orders", strings.NewReader(`{
				"clientRequestId":"11111111-1111-4111-8111-111111111111",
				"paymentMethod":"cash",
				"lines":[{"menuItemSlug":"kopi-susu","quantity":1,"modifiers":[]}]
			}`))
			request.AddCookie(cookie)
			response := httptest.NewRecorder()

			fixture.router.ServeHTTP(response, request)

			if test.wantCode != "" {
				assertJSONError(t, response, test.wantStatus, test.wantCode)
				return
			}
			if response.Code != test.wantStatus {
				t.Fatalf("status = %d, want %d", response.Code, test.wantStatus)
			}
		})
	}
}

func TestCancelPaidOrderRejectsMalformedOrderID(t *testing.T) {
	fixture := newOrderRouterFixture(t, fakeOrderService{})
	cookie := assertSingleCookie(t, fixture.login(t), fixture.cookieName)
	request := httptest.NewRequest(http.MethodPost, "/api/pos/orders/001/cancel", nil)
	request.AddCookie(cookie)
	response := httptest.NewRecorder()

	fixture.router.ServeHTTP(response, request)

	assertJSONError(t, response, http.StatusBadRequest, "invalid_order_id")
}

func TestCancelPaidOrderReturnsUpdatedDetail(t *testing.T) {
	cancelledAt := time.Date(2026, 6, 30, 11, 0, 0, 0, time.UTC)
	service := fakeOrderService{cancelDetail: apporders.PaidOrderDetail{
		OrderID:       "1",
		QueueNumber:   1,
		BusinessDate:  "2026-06-30",
		Status:        "cancelled",
		PaymentMethod: "cash",
		PaidAt:        time.Date(2026, 6, 30, 10, 0, 0, 0, time.UTC),
		CancelledAt:   &cancelledAt,
		TotalRp:       18000,
	}}
	fixture := newOrderRouterFixture(t, service)
	cookie := assertSingleCookie(t, fixture.login(t), fixture.cookieName)
	request := httptest.NewRequest(http.MethodPost, "/api/pos/orders/1/cancel", nil)
	request.AddCookie(cookie)
	response := httptest.NewRecorder()

	fixture.router.ServeHTTP(response, request)

	if response.Code != http.StatusOK {
		t.Fatalf("status = %d, want 200", response.Code)
	}
}

func TestCancelPaidOrderMapsServiceErrors(t *testing.T) {
	tests := []struct {
		name       string
		service    fakeOrderService
		wantStatus int
		wantCode   string
	}{
		{name: "not found", service: fakeOrderService{cancelErr: apporders.ErrOrderNotFound}, wantStatus: http.StatusNotFound, wantCode: "not_found"},
		{name: "not cancellable", service: fakeOrderService{cancelErr: apporders.ErrOrderNotCancellable}, wantStatus: http.StatusConflict, wantCode: "order_not_cancellable"},
		{name: "internal", service: fakeOrderService{cancelErr: errors.New("database unavailable")}, wantStatus: http.StatusInternalServerError, wantCode: "internal_error"},
	}
	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			fixture := newOrderRouterFixture(t, test.service)
			cookie := assertSingleCookie(t, fixture.login(t), fixture.cookieName)
			request := httptest.NewRequest(http.MethodPost, "/api/pos/orders/1/cancel", nil)
			request.AddCookie(cookie)
			response := httptest.NewRecorder()

			fixture.router.ServeHTTP(response, request)

			assertJSONError(t, response, test.wantStatus, test.wantCode)
		})
	}
}

type orderRouterFixture struct {
	router     http.Handler
	cookieName string
}

func newOrderRouterFixture(t *testing.T, orderService fakeOrderService) orderRouterFixture {
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
	menuService := appmenu.NewService(appmenu.Dependencies{Repository: &fakeOrderHTTPMenuRepository{}})
	cookieName := "coffee_pos_session"
	return orderRouterFixture{
		router: NewRouter(RouterOptions{
			AuthService:  authService,
			MenuService:  &menuService,
			OrderService: orderService,
			Cookie:       CookieConfig{Name: cookieName, Path: "/", SameSite: http.SameSiteLaxMode},
		}),
		cookieName: cookieName,
	}
}

func (fixture orderRouterFixture) login(t *testing.T) *httptest.ResponseRecorder {
	t.Helper()
	request := httptest.NewRequest(http.MethodPost, "/api/auth/login", strings.NewReader(`{"pin":"123456"}`))
	request.RemoteAddr = "203.0.113.10:1234"
	response := httptest.NewRecorder()
	fixture.router.ServeHTTP(response, request)
	return response
}

func minimalPaidOrderDetail() apporders.PaidOrderDetail {
	return apporders.PaidOrderDetail{
		OrderID:       "1",
		QueueNumber:   1,
		BusinessDate:  "2026-06-30",
		Status:        "paid",
		PaymentMethod: "cash",
		PaidAt:        time.Date(2026, 6, 30, 10, 0, 0, 0, time.UTC),
		TotalRp:       18000,
	}
}

type fakeOrderService struct {
	detail       apporders.PaidOrderDetail
	result       apporders.CreatePaidOrderResult
	err          error
	cancelDetail apporders.PaidOrderDetail
	cancelResult apporders.CancelPaidOrderResult
	cancelErr    error
}

func (service fakeOrderService) CreatePaidOrder(context.Context, apporders.CreatePaidOrderInput) (apporders.PaidOrderDetail, apporders.CreatePaidOrderResult, error) {
	result := service.result
	if result == "" {
		result = apporders.CreatePaidOrderCreated
	}
	return service.detail, result, service.err
}

func (service fakeOrderService) CancelPaidOrder(context.Context, apporders.CancelPaidOrderInput) (apporders.PaidOrderDetail, apporders.CancelPaidOrderResult, error) {
	result := service.cancelResult
	if result == "" {
		result = apporders.CancelPaidOrderCancelled
	}
	return service.cancelDetail, result, service.cancelErr
}

type fakeOrderHTTPMenuRepository struct{}

func (repo *fakeOrderHTTPMenuRepository) SeedMenu(context.Context, domainmenu.Seed) error {
	return nil
}

func (repo *fakeOrderHTTPMenuRepository) GetCashierMenu(context.Context) (appmenu.CashierMenu, error) {
	return appmenu.CashierMenu{}, nil
}
