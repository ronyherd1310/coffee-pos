package http

import (
	"encoding/json"
	"errors"
	"io"
	"net/http"
	"time"

	apporders "coffee-pos/backend/internal/app/orders"
)

const createOrderBodyLimit = 16 * 1024

var forbiddenCreateOrderFields = map[string]struct{}{
	"orderId":      {},
	"queueNumber":  {},
	"businessDate": {},
	"status":       {},
	"paidAt":       {},
	"cancelledAt":  {},
	"totalRp":      {},
	"lineTotalRp":  {},
	"unitPriceRp":  {},
}

type orderHandlers struct {
	service orderService
}

func newOrderHandlers(service orderService) orderHandlers {
	return orderHandlers{service: service}
}

func (handler orderHandlers) handleCreatePaidOrder(w http.ResponseWriter, r *http.Request) {
	if handler.service == nil {
		writeJSONError(w, http.StatusInternalServerError, "internal_error")
		return
	}
	input, ok := readCreateOrderInput(w, r)
	if !ok {
		return
	}
	detail, result, err := handler.service.CreatePaidOrder(r.Context(), input)
	if err != nil {
		switch {
		case errors.Is(err, apporders.ErrInvalidClientRequestID):
			writeJSONError(w, http.StatusBadRequest, "invalid_client_request_id")
		case errors.Is(err, apporders.ErrInvalidOrder):
			writeJSONError(w, http.StatusUnprocessableEntity, "invalid_order")
		case errors.Is(err, apporders.ErrIdempotencyConflict):
			writeJSONError(w, http.StatusConflict, "idempotency_conflict")
		default:
			writeJSONError(w, http.StatusInternalServerError, "internal_error")
		}
		return
	}
	status := http.StatusCreated
	if result == apporders.CreatePaidOrderExisting {
		status = http.StatusOK
	}
	writeJSON(w, status, paidOrderDetailResponseFromApp(detail))
}

func (handler orderHandlers) handleCancelPaidOrder(w http.ResponseWriter, r *http.Request) {
	if handler.service == nil {
		writeJSONError(w, http.StatusInternalServerError, "internal_error")
		return
	}
	orderID := r.PathValue("orderId")
	if !validPathOrderID(orderID) {
		writeJSONError(w, http.StatusBadRequest, "invalid_order_id")
		return
	}
	detail, _, err := handler.service.CancelPaidOrder(r.Context(), apporders.CancelPaidOrderInput{OrderID: orderID})
	if err != nil {
		switch {
		case errors.Is(err, apporders.ErrInvalidOrderID):
			writeJSONError(w, http.StatusBadRequest, "invalid_order_id")
		case errors.Is(err, apporders.ErrOrderNotFound):
			writeJSONError(w, http.StatusNotFound, "not_found")
		case errors.Is(err, apporders.ErrOrderNotCancellable):
			writeJSONError(w, http.StatusConflict, "order_not_cancellable")
		default:
			writeJSONError(w, http.StatusInternalServerError, "internal_error")
		}
		return
	}
	writeJSON(w, http.StatusOK, paidOrderDetailResponseFromApp(detail))
}

func readCreateOrderInput(w http.ResponseWriter, r *http.Request) (apporders.CreatePaidOrderInput, bool) {
	reader := http.MaxBytesReader(w, r.Body, createOrderBodyLimit)
	defer reader.Close()

	var raw map[string]json.RawMessage
	decoder := json.NewDecoder(reader)
	if err := decoder.Decode(&raw); err != nil {
		writeJSONError(w, http.StatusBadRequest, "invalid_json")
		return apporders.CreatePaidOrderInput{}, false
	}
	for field := range raw {
		if _, forbidden := forbiddenCreateOrderFields[field]; forbidden {
			writeJSONError(w, http.StatusBadRequest, "forbidden_field")
			return apporders.CreatePaidOrderInput{}, false
		}
		switch field {
		case "clientRequestId", "paymentMethod", "note", "lines":
		default:
			writeJSONError(w, http.StatusBadRequest, "unknown_field")
			return apporders.CreatePaidOrderInput{}, false
		}
	}
	var extra any
	if err := decoder.Decode(&extra); !errors.Is(err, io.EOF) {
		writeJSONError(w, http.StatusBadRequest, "invalid_json")
		return apporders.CreatePaidOrderInput{}, false
	}

	input, err := decodeCreateOrderRaw(raw)
	if err != nil {
		writeJSONError(w, http.StatusBadRequest, err.Error())
		return apporders.CreatePaidOrderInput{}, false
	}
	return input, true
}

func decodeCreateOrderRaw(raw map[string]json.RawMessage) (apporders.CreatePaidOrderInput, error) {
	clientRequestID, err := requiredString(raw, "clientRequestId")
	if err != nil {
		return apporders.CreatePaidOrderInput{}, err
	}
	paymentMethod, err := requiredString(raw, "paymentMethod")
	if err != nil {
		return apporders.CreatePaidOrderInput{}, err
	}
	var note *string
	if value, ok := raw["note"]; ok {
		if string(value) == "null" {
			return apporders.CreatePaidOrderInput{}, errors.New("invalid_field_type")
		}
		var decoded string
		if err := json.Unmarshal(value, &decoded); err != nil {
			return apporders.CreatePaidOrderInput{}, errors.New("invalid_field_type")
		}
		note = &decoded
	}
	linesRaw, ok := raw["lines"]
	if !ok {
		return apporders.CreatePaidOrderInput{}, errors.New("missing_field")
	}
	var rawLines []map[string]json.RawMessage
	if err := json.Unmarshal(linesRaw, &rawLines); err != nil || rawLines == nil {
		return apporders.CreatePaidOrderInput{}, errors.New("invalid_field_type")
	}
	if len(rawLines) == 0 {
		return apporders.CreatePaidOrderInput{}, errors.New("missing_field")
	}
	lines := make([]apporders.CreatePaidOrderLineInput, 0, len(rawLines))
	for _, rawLine := range rawLines {
		line, err := decodeCreateOrderLineRaw(rawLine)
		if err != nil {
			return apporders.CreatePaidOrderInput{}, err
		}
		lines = append(lines, line)
	}
	return apporders.CreatePaidOrderInput{
		ClientRequestID: clientRequestID,
		PaymentMethod:   paymentMethod,
		Note:            note,
		Lines:           lines,
	}, nil
}

func decodeCreateOrderLineRaw(raw map[string]json.RawMessage) (apporders.CreatePaidOrderLineInput, error) {
	for field := range raw {
		if _, forbidden := forbiddenCreateOrderFields[field]; forbidden {
			return apporders.CreatePaidOrderLineInput{}, errors.New("forbidden_field")
		}
		switch field {
		case "menuItemSlug", "quantity", "modifiers":
		default:
			return apporders.CreatePaidOrderLineInput{}, errors.New("unknown_field")
		}
	}
	menuItemSlug, err := requiredString(raw, "menuItemSlug")
	if err != nil {
		return apporders.CreatePaidOrderLineInput{}, err
	}
	var quantity int
	if value, ok := raw["quantity"]; !ok {
		return apporders.CreatePaidOrderLineInput{}, errors.New("missing_field")
	} else if err := json.Unmarshal(value, &quantity); err != nil {
		return apporders.CreatePaidOrderLineInput{}, errors.New("invalid_field_type")
	}
	if quantity < 1 || quantity > 99 {
		return apporders.CreatePaidOrderLineInput{}, errors.New("invalid_field_type")
	}
	var rawModifiers []map[string]json.RawMessage
	if value, ok := raw["modifiers"]; !ok {
		return apporders.CreatePaidOrderLineInput{}, errors.New("missing_field")
	} else if err := json.Unmarshal(value, &rawModifiers); err != nil || rawModifiers == nil {
		return apporders.CreatePaidOrderLineInput{}, errors.New("invalid_field_type")
	}
	modifiers := make([]apporders.CreatePaidOrderModifierInput, 0, len(rawModifiers))
	for _, rawModifier := range rawModifiers {
		modifier, err := decodeCreateOrderModifierRaw(rawModifier)
		if err != nil {
			return apporders.CreatePaidOrderLineInput{}, err
		}
		modifiers = append(modifiers, modifier)
	}
	return apporders.CreatePaidOrderLineInput{MenuItemSlug: menuItemSlug, Quantity: quantity, Modifiers: modifiers}, nil
}

func decodeCreateOrderModifierRaw(raw map[string]json.RawMessage) (apporders.CreatePaidOrderModifierInput, error) {
	for field := range raw {
		if _, forbidden := forbiddenCreateOrderFields[field]; forbidden {
			return apporders.CreatePaidOrderModifierInput{}, errors.New("forbidden_field")
		}
		switch field {
		case "groupSlug", "optionSlug":
		default:
			return apporders.CreatePaidOrderModifierInput{}, errors.New("unknown_field")
		}
	}
	groupSlug, err := requiredString(raw, "groupSlug")
	if err != nil {
		return apporders.CreatePaidOrderModifierInput{}, err
	}
	optionSlug, err := requiredString(raw, "optionSlug")
	if err != nil {
		return apporders.CreatePaidOrderModifierInput{}, err
	}
	return apporders.CreatePaidOrderModifierInput{GroupSlug: groupSlug, OptionSlug: optionSlug}, nil
}

func requiredString(raw map[string]json.RawMessage, field string) (string, error) {
	value, ok := raw[field]
	if !ok {
		return "", errors.New("missing_field")
	}
	if string(value) == "null" {
		return "", errors.New("invalid_field_type")
	}
	var decoded string
	if err := json.Unmarshal(value, &decoded); err != nil {
		return "", errors.New("invalid_field_type")
	}
	return decoded, nil
}

type paidOrderDetailResponse struct {
	OrderID       string                  `json:"orderId"`
	QueueNumber   int                     `json:"queueNumber"`
	BusinessDate  string                  `json:"businessDate"`
	Status        string                  `json:"status"`
	PaymentMethod string                  `json:"paymentMethod"`
	PaidAt        string                  `json:"paidAt"`
	CancelledAt   *string                 `json:"cancelledAt"`
	Note          *string                 `json:"note"`
	TotalRp       int64                   `json:"totalRp"`
	Lines         []paidOrderLineResponse `json:"lines"`
}

type paidOrderLineResponse struct {
	MenuItemSlug string                      `json:"menuItemSlug"`
	MenuItemName string                      `json:"menuItemName"`
	UnitPriceRp  int64                       `json:"unitPriceRp"`
	Quantity     int                         `json:"quantity"`
	LineTotalRp  int64                       `json:"lineTotalRp"`
	Modifiers    []paidOrderModifierResponse `json:"modifiers"`
}

type paidOrderModifierResponse struct {
	GroupSlug    string `json:"groupSlug"`
	GroupName    string `json:"groupName"`
	OptionSlug   string `json:"optionSlug"`
	OptionName   string `json:"optionName"`
	PriceDeltaRp int64  `json:"priceDeltaRp"`
}

func paidOrderDetailResponseFromApp(detail apporders.PaidOrderDetail) paidOrderDetailResponse {
	response := paidOrderDetailResponse{
		OrderID:       detail.OrderID,
		QueueNumber:   detail.QueueNumber,
		BusinessDate:  detail.BusinessDate,
		Status:        detail.Status,
		PaymentMethod: detail.PaymentMethod,
		PaidAt:        detail.PaidAt.Format(time.RFC3339),
		CancelledAt:   nullableRFC3339(detail.CancelledAt),
		Note:          detail.Note,
		TotalRp:       detail.TotalRp,
		Lines:         make([]paidOrderLineResponse, 0, len(detail.Lines)),
	}
	for _, line := range detail.Lines {
		lineResponse := paidOrderLineResponse{
			MenuItemSlug: line.MenuItemSlug,
			MenuItemName: line.MenuItemName,
			UnitPriceRp:  line.UnitPriceRp,
			Quantity:     line.Quantity,
			LineTotalRp:  line.LineTotalRp,
			Modifiers:    make([]paidOrderModifierResponse, 0, len(line.Modifiers)),
		}
		for _, modifier := range line.Modifiers {
			lineResponse.Modifiers = append(lineResponse.Modifiers, paidOrderModifierResponse{
				GroupSlug:    modifier.GroupSlug,
				GroupName:    modifier.GroupName,
				OptionSlug:   modifier.OptionSlug,
				OptionName:   modifier.OptionName,
				PriceDeltaRp: modifier.PriceDeltaRp,
			})
		}
		response.Lines = append(response.Lines, lineResponse)
	}
	return response
}

func nullableRFC3339(value *time.Time) *string {
	if value == nil {
		return nil
	}
	formatted := value.Format(time.RFC3339)
	return &formatted
}

func validPathOrderID(value string) bool {
	if value == "" || value[0] == '0' {
		return false
	}
	for _, char := range value {
		if char < '0' || char > '9' {
			return false
		}
	}
	return true
}
