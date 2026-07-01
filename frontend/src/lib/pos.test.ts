import { afterEach, describe, expect, it, vi } from "vitest";
import { cancelPaidOrder, createPaidOrder, getCashierMenu } from "./pos";

const menuResponse = {
  categories: [
    {
      name: "Coffee",
      slug: "coffee",
      items: [
        {
          name: "Americano",
          slug: "americano",
          priceRp: 18000,
          modifierGroups: [
            {
              name: "Temperature",
              slug: "temperature",
              required: true,
              selectionType: "single",
              options: [{ name: "Hot", slug: "hot", priceDeltaRp: 0 }]
            }
          ]
        }
      ]
    }
  ]
};

const paidOrderResponse = {
  orderId: "order-1",
  queueNumber: 1,
  businessDate: "2026-07-01",
  status: "paid",
  paymentMethod: "cash",
  paidAt: "2026-07-01T03:00:00Z",
  cancelledAt: null,
  note: "Less ice",
  totalRp: 18000,
  lines: [
    {
      menuItemSlug: "americano",
      menuItemName: "Americano",
      unitPriceRp: 18000,
      quantity: 1,
      lineTotalRp: 18000,
      modifiers: [
        {
          groupSlug: "temperature",
          groupName: "Temperature",
          optionSlug: "hot",
          optionName: "Hot",
          priceDeltaRp: 0
        }
      ]
    }
  ]
};

describe("cashier POS API client", () => {
  afterEach(() => {
    vi.unstubAllGlobals();
  });

  it("loads and validates the cashier menu with same-origin credentials", async () => {
    const fetchMock = vi.fn().mockResolvedValue(jsonResponse(menuResponse));
    vi.stubGlobal("fetch", fetchMock);

    await expect(getCashierMenu()).resolves.toEqual({ status: "success", menu: menuResponse });
    expect(fetchMock).toHaveBeenCalledWith("/api/pos/menu", {
      credentials: "same-origin",
      headers: { Accept: "application/json" },
      method: "GET"
    });
  });

  it("maps unauthorized menu responses", async () => {
    vi.stubGlobal(
      "fetch",
      vi.fn().mockResolvedValue(jsonResponse({ error: "unauthorized" }, { status: 401 }))
    );

    await expect(getCashierMenu()).resolves.toEqual({ status: "unauthorized" });
  });

  it("maps malformed menu JSON to an unexpected result", async () => {
    vi.stubGlobal("fetch", vi.fn().mockResolvedValue(jsonResponse({ categories: "bad" })));

    await expect(getCashierMenu()).resolves.toEqual({ status: "unexpected" });
  });

  it("maps network failures while loading the menu to unavailable", async () => {
    vi.stubGlobal("fetch", vi.fn().mockRejectedValue(new TypeError("network")));

    await expect(getCashierMenu()).resolves.toEqual({ status: "unavailable" });
  });

  it("creates a paid order without adding server-owned fields", async () => {
    const fetchMock = vi.fn().mockResolvedValue(jsonResponse(paidOrderResponse, { status: 201 }));
    vi.stubGlobal("fetch", fetchMock);

    const input = {
      clientRequestId: "11111111-1111-4111-8111-111111111111",
      paymentMethod: "cash" as const,
      note: "Less ice",
      lines: [
        {
          menuItemSlug: "americano",
          quantity: 1,
          modifiers: [{ groupSlug: "temperature", optionSlug: "hot" }]
        }
      ]
    };

    await expect(createPaidOrder(input)).resolves.toEqual({
      status: "success",
      order: paidOrderResponse
    });
    expect(fetchMock).toHaveBeenCalledWith("/api/pos/orders", {
      body: JSON.stringify(input),
      credentials: "same-origin",
      headers: {
        Accept: "application/json",
        "Content-Type": "application/json"
      },
      method: "POST"
    });
  });

  it("treats an idempotent paid-order retry success as success", async () => {
    vi.stubGlobal("fetch", vi.fn().mockResolvedValue(jsonResponse(paidOrderResponse, { status: 200 })));

    await expect(
      createPaidOrder({
        clientRequestId: "11111111-1111-4111-8111-111111111111",
        paymentMethod: "cash",
        lines: []
      })
    ).resolves.toEqual({ status: "success", order: paidOrderResponse });
  });

  it.each([
    [400, "invalid_client_request_id", "invalid-client-request-id"],
    [401, "unauthorized", "unauthorized"],
    [409, "idempotency_conflict", "idempotency-conflict"],
    [422, "invalid_order", "invalid-order"]
  ] as const)("maps create-order %s %s errors", async (status, error, expectedStatus) => {
    vi.stubGlobal("fetch", vi.fn().mockResolvedValue(jsonResponse({ error }, { status })));

    await expect(
      createPaidOrder({
        clientRequestId: "bad",
        paymentMethod: "cash",
        lines: []
      })
    ).resolves.toEqual({ status: expectedStatus });
  });

  it("maps malformed create-order JSON to unexpected", async () => {
    vi.stubGlobal("fetch", vi.fn().mockResolvedValue(jsonResponse({ orderId: 123 }, { status: 201 })));

    await expect(
      createPaidOrder({
        clientRequestId: "11111111-1111-4111-8111-111111111111",
        paymentMethod: "cash",
        lines: []
      })
    ).resolves.toEqual({ status: "unexpected" });
  });

  it("maps network failures while creating orders to unavailable", async () => {
    vi.stubGlobal("fetch", vi.fn().mockRejectedValue(new TypeError("network")));

    await expect(
      createPaidOrder({
        clientRequestId: "11111111-1111-4111-8111-111111111111",
        paymentMethod: "cash",
        lines: []
      })
    ).resolves.toEqual({ status: "unavailable" });
  });

  it("cancels a paid order by id", async () => {
    const cancelled = { ...paidOrderResponse, status: "cancelled", cancelledAt: "2026-07-01T04:00:00Z" };
    const fetchMock = vi.fn().mockResolvedValue(jsonResponse(cancelled));
    vi.stubGlobal("fetch", fetchMock);

    await expect(cancelPaidOrder("order-1")).resolves.toEqual({
      status: "success",
      order: cancelled
    });
    expect(fetchMock).toHaveBeenCalledWith("/api/pos/orders/order-1/cancel", {
      credentials: "same-origin",
      headers: { Accept: "application/json" },
      method: "POST"
    });
  });

  it.each([
    [401, "unauthorized", "unauthorized"],
    [404, "not_found", "not-found"],
    [409, "order_not_cancellable", "not-cancellable"]
  ] as const)("maps cancel-order %s %s errors", async (status, error, expectedStatus) => {
    vi.stubGlobal("fetch", vi.fn().mockResolvedValue(jsonResponse({ error }, { status })));

    await expect(cancelPaidOrder("order-1")).resolves.toEqual({ status: expectedStatus });
  });

  it("maps malformed cancellation JSON to unexpected", async () => {
    vi.stubGlobal("fetch", vi.fn().mockResolvedValue(jsonResponse({ status: "cancelled" })));

    await expect(cancelPaidOrder("order-1")).resolves.toEqual({ status: "unexpected" });
  });
});

function jsonResponse(body: unknown, init: ResponseInit = {}) {
  return new Response(JSON.stringify(body), {
    headers: { "Content-Type": "application/json" },
    status: 200,
    ...init
  });
}
