import { fireEvent, render, screen, waitFor, within } from "@testing-library/preact";
import { afterEach, describe, expect, it, vi } from "vitest";
import { CashierOrderScreen } from "./CashierOrderScreen";

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
          modifierGroups: []
        }
      ]
    }
  ]
};

const orderMenuResponse = {
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
              options: [
                { name: "Hot", slug: "hot", priceDeltaRp: 0 },
                { name: "Iced", slug: "iced", priceDeltaRp: 2000 }
              ]
            },
            {
              name: "Sugar",
              slug: "sugar",
              required: true,
              selectionType: "single",
              options: [
                { name: "Normal", slug: "normal", priceDeltaRp: 0 },
                { name: "Less", slug: "less", priceDeltaRp: 0 }
              ]
            }
          ]
        },
        {
          name: "Latte",
          slug: "latte",
          priceRp: 25000,
          modifierGroups: [
            {
              name: "Temperature",
              slug: "temperature",
              required: true,
              selectionType: "single",
              options: [
                { name: "Hot", slug: "hot", priceDeltaRp: 0 },
                { name: "Iced", slug: "iced", priceDeltaRp: 2000 }
              ]
            },
            {
              name: "Sugar",
              slug: "sugar",
              required: true,
              selectionType: "single",
              options: [
                { name: "Normal", slug: "normal", priceDeltaRp: 0 },
                { name: "Less", slug: "less", priceDeltaRp: 0 }
              ]
            }
          ]
        }
      ]
    }
  ]
};

const revampMenuResponse = {
  categories: [
    {
      ...orderMenuResponse.categories[0],
      items: orderMenuResponse.categories[0].items.map((item, index) => ({
        ...item,
        imagePath: index === 0 ? "/menu/americano.png" : "/menu/latte.png",
        popularityRank: index + 1,
        bestSeller: true,
        iced: true,
        lowSugar: true,
        newArrival: false
      }))
    },
    {
      name: "Tea",
      slug: "tea",
      items: [
        {
          name: "Iced Tea",
          slug: "iced-tea",
          priceRp: 15000,
          imagePath: "/menu/iced-tea.png",
          popularityRank: 3,
          iced: true,
          modifierGroups: []
        }
      ]
    },
    {
      name: "Snacks",
      slug: "snacks",
      items: [
        {
          name: "Muffin",
          slug: "muffin",
          priceRp: 20000,
          imagePath: "/menu/muffin.png",
          popularityRank: 4,
          newArrival: true,
          modifierGroups: []
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
        },
        {
          groupSlug: "sugar",
          groupName: "Sugar",
          optionSlug: "normal",
          optionName: "Normal",
          priceDeltaRp: 0
        }
      ]
    }
  ]
};

describe("CashierOrderScreen menu loading", () => {
  afterEach(() => {
    vi.unstubAllGlobals();
  });

  it("shows a loading state while fetching the cashier menu", () => {
    vi.stubGlobal("fetch", vi.fn().mockReturnValue(new Promise(() => undefined)));

    render(<CashierOrderScreen onSessionExpired={vi.fn()} />);

    expect(screen.getByRole("status")).toHaveTextContent("Loading menu...");
  });

  it("renders backend menu items after a successful load", async () => {
    vi.stubGlobal("fetch", vi.fn().mockResolvedValue(jsonResponse(menuResponse)));

    render(<CashierOrderScreen onSessionExpired={vi.fn()} />);

    expect(await screen.findByRole("button", { name: /Americano/ })).toBeVisible();
    expect(screen.getByRole("heading", { level: 2, name: "New Order" })).toBeVisible();
  });

  it("shows a retryable unavailable state when the menu request fails", async () => {
    const fetchMock = vi
      .fn()
      .mockResolvedValueOnce(jsonResponse({ error: "internal_error" }, { status: 503 }))
      .mockResolvedValueOnce(jsonResponse(menuResponse));
    vi.stubGlobal("fetch", fetchMock);

    render(<CashierOrderScreen onSessionExpired={vi.fn()} />);

    expect(await screen.findByRole("alert")).toHaveTextContent("Cannot load the cashier menu.");

    fireEvent.click(screen.getByRole("button", { name: "Retry menu" }));

    expect(await screen.findByRole("button", { name: /Americano/ })).toBeVisible();
  });

  it("hands unauthorized menu responses back to auth handling", async () => {
    const onSessionExpired = vi.fn();
    vi.stubGlobal(
      "fetch",
      vi.fn().mockResolvedValue(jsonResponse({ error: "unauthorized" }, { status: 401 }))
    );

    render(<CashierOrderScreen onSessionExpired={onSessionExpired} />);

    await waitFor(() => expect(onSessionExpired).toHaveBeenCalledTimes(1));
  });

  it("shows an empty state for a valid menu with no items", async () => {
    vi.stubGlobal("fetch", vi.fn().mockResolvedValue(jsonResponse({ categories: [] })));

    render(<CashierOrderScreen onSessionExpired={vi.fn()} />);

    await waitFor(() =>
      expect(screen.getByRole("status")).toHaveTextContent("No menu items available.")
    );
  });
});

describe("CashierOrderScreen draft order flow", () => {
  afterEach(() => {
    vi.unstubAllGlobals();
  });

  it("requires modifiers before adding a selected menu item and respects pre-add quantity bounds", async () => {
    vi.stubGlobal("fetch", vi.fn().mockResolvedValue(jsonResponse(orderMenuResponse)));

    render(<CashierOrderScreen onSessionExpired={vi.fn()} />);

    fireEvent.click(await screen.findByRole("button", { name: /Americano/ }));

    expect(screen.getByRole("heading", { level: 3, name: "Configure Americano" })).toBeVisible();
    expect(screen.getByRole("button", { name: "Add Item To Order" })).toBeDisabled();
    expect(screen.getByLabelText("Selected item quantity")).toHaveValue(1);

    fireEvent.click(screen.getByRole("button", { name: "Decrease selected item quantity" }));
    expect(screen.getByLabelText("Selected item quantity")).toHaveValue(1);

    fireEvent.click(screen.getByLabelText("Iced +Rp2.000"));
    fireEvent.click(screen.getByLabelText("Less"));
    fireEvent.click(screen.getByRole("button", { name: "Increase selected item quantity" }));
    fireEvent.click(screen.getByRole("button", { name: "Add Item To Order" }));

    expect(screen.getByText("Americano", { selector: ".cart-line__name" })).toBeVisible();
    expect(screen.getByText("Iced, Less")).toBeVisible();
    expect(screen.getByText("Rp40.000", { selector: ".cart-line p" })).toBeVisible();
  });

  it("adds the same drink with different modifiers as separate cart lines", async () => {
    vi.stubGlobal("fetch", vi.fn().mockResolvedValue(jsonResponse(orderMenuResponse)));

    render(<CashierOrderScreen onSessionExpired={vi.fn()} />);

    await addAmericanoLine("Hot", "Normal");
    await addAmericanoLine("Iced +Rp2.000", "Less");

    expect(screen.getByText("Hot, Normal")).toBeVisible();
    expect(screen.getByText("Iced, Less")).toBeVisible();
    expect(screen.getAllByText("Americano", { selector: ".cart-line__name" })).toHaveLength(2);
  });

  it("updates cart quantities, removes lines, and recalculates totals", async () => {
    vi.stubGlobal("fetch", vi.fn().mockResolvedValue(jsonResponse(orderMenuResponse)));

    render(<CashierOrderScreen onSessionExpired={vi.fn()} />);

    await addAmericanoLine("Hot", "Normal");

    expect(screen.getByText("Total")).toBeVisible();
    expect(screen.getByText("Tax")).toBeVisible();
    expect(screen.getByText("Rp1.980", { selector: ".order-total__tax dd" })).toBeVisible();
    expect(screen.getByText("Rp19.980", { selector: ".payment-total" })).toBeVisible();

    fireEvent.click(screen.getByRole("button", { name: "Increase Americano quantity" }));
    expect(screen.getByText("Rp39.960", { selector: ".payment-total" })).toBeVisible();

    fireEvent.click(screen.getByRole("button", { name: "Decrease Americano quantity" }));
    expect(screen.getByText("Rp19.980", { selector: ".payment-total" })).toBeVisible();

    fireEvent.click(screen.getByRole("button", { name: "Remove Americano" }));
    expect(screen.getByText("No items added yet.")).toBeVisible();
    expect(screen.getByText("Rp0", { selector: ".payment-total" })).toBeVisible();
  });

  it("renders current order rows with thumbnails and stable controls", async () => {
    vi.stubGlobal("fetch", vi.fn().mockResolvedValue(jsonResponse(revampMenuResponse)));

    render(<CashierOrderScreen onSessionExpired={vi.fn()} />);

    await addAmericanoLine("Hot", "Normal");

    const row = screen.getByText("Americano", { selector: ".cart-line__name" }).closest("li") as HTMLElement;
    expect(row.querySelector(".cart-line__thumb img")).toHaveAttribute("src", "/menu/americano.png");
    expect(within(row).getByText("1x")).toBeVisible();
    expect(within(row).getByText("Americano")).toBeVisible();
    expect(within(row).getByText("Hot, Normal")).toBeVisible();
    expect(within(row).getByText("Rp18.000")).toBeVisible();
    expect(within(row).getByRole("button", { name: "Increase Americano quantity" })).toBeVisible();
    expect(within(row).getByRole("button", { name: "Remove Americano" })).toBeVisible();
  });

  it("limits order notes to 120 characters and shows the note count", async () => {
    vi.stubGlobal("fetch", vi.fn().mockResolvedValue(jsonResponse(orderMenuResponse)));

    render(<CashierOrderScreen onSessionExpired={vi.fn()} />);

    const note = await screen.findByLabelText("Order note");
    fireEvent.input(note, { target: { value: "a".repeat(130) } });

    expect(note).toHaveValue("a".repeat(120));
    expect(screen.getByText("120 / 120")).toBeVisible();
  });

  it("selects payment method, shows QRIS, and gates unpaid actions", async () => {
    vi.stubGlobal("fetch", vi.fn().mockResolvedValue(jsonResponse(orderMenuResponse)));

    render(<CashierOrderScreen onSessionExpired={vi.fn()} />);

    expect(await screen.findByRole("button", { name: /Americano/ })).toBeVisible();
    expect(screen.getByRole("button", { name: "Proceed to Payment" })).toBeDisabled();
    expect(screen.queryByRole("button", { name: "Print Ticket" })).not.toBeInTheDocument();

    await addAmericanoLine("Hot", "Normal");

    expect(screen.getByRole("button", { name: "Proceed to Payment" })).toBeDisabled();

    fireEvent.click(screen.getByLabelText("QRIS"));

    expect(screen.getByRole("button", { name: "Proceed to Payment" })).toBeEnabled();
    expect(screen.queryByAltText("Static QRIS payment code")).not.toBeInTheDocument();
  });

  it("renders catalog controls and preserves draft state when controls change", async () => {
    vi.stubGlobal("fetch", vi.fn().mockResolvedValue(jsonResponse(revampMenuResponse)));

    const { container } = render(<CashierOrderScreen onSessionExpired={vi.fn()} />);

    await addAmericanoLine("Hot", "Normal");
    fireEvent.click(await screen.findByRole("button", { name: /Latte/ }));
    fireEvent.click(screen.getByLabelText("Iced +Rp2.000"));
    fireEvent.click(screen.getByLabelText("Less"));
    fireEvent.input(screen.getByLabelText("Order note"), { target: { value: "Keep hot line" } });
    fireEvent.click(screen.getByLabelText("QRIS"));

    expect(screen.getByPlaceholderText("Search menu item...")).toBeVisible();
    expect(screen.getByRole("tab", { name: "All" })).toBeVisible();
    expect(screen.getByRole("tab", { name: "Coffee" })).toBeVisible();
    expect(screen.getByRole("button", { name: "Best Seller" })).toBeVisible();
    expect(screen.getByRole("button", { name: "Iced" })).toBeVisible();
    expect(screen.getByRole("button", { name: "Low Sugar" })).toBeVisible();
    expect(screen.getByRole("button", { name: "New Arrival" })).toBeVisible();
    expect(screen.getByLabelText("Sort menu")).toHaveValue("popular");
    expect(screen.queryByRole("button", { name: /list view/i })).not.toBeInTheDocument();
    expect(container.querySelector(".catalog-view-toggle")).not.toBeInTheDocument();

    fireEvent.input(screen.getByPlaceholderText("Search menu item..."), { target: { value: "tea" } });
    fireEvent.click(screen.getByRole("tab", { name: "Tea" }));
    fireEvent.click(screen.getByRole("button", { name: "Iced" }));
    fireEvent.change(screen.getByLabelText("Sort menu"), { target: { value: "popular" } });

    expect(screen.getByText("Hot, Normal")).toBeVisible();
    expect(screen.getByLabelText("Order note")).toHaveValue("Keep hot line");
    expect(screen.getByLabelText("QRIS")).toBeChecked();
    expect(screen.getByRole("heading", { level: 3, name: "Configure Latte" })).toBeVisible();
    expect(screen.getByLabelText("Iced +Rp2.000")).toBeChecked();
    expect(screen.getByLabelText("Less")).toBeChecked();
  });

  it("renders filtered menu results as image cards with badges and empty state", async () => {
    vi.stubGlobal("fetch", vi.fn().mockResolvedValue(jsonResponse(revampMenuResponse)));

    render(<CashierOrderScreen onSessionExpired={vi.fn()} />);

    const americanoCard = await screen.findByRole("button", { name: /Americano/ });
    expect(americanoCard.querySelector("img")).toHaveAttribute("src", "/menu/americano.png");

    fireEvent.input(screen.getByPlaceholderText("Search menu item..."), { target: { value: "iced tea" } });
    const icedTeaCard = screen.getByRole("button", { name: /Iced Tea/ });
    expect(icedTeaCard.querySelector("img")).toHaveAttribute("src", "/menu/iced-tea.png");

    fireEvent.input(screen.getByPlaceholderText("Search menu item..."), { target: { value: "no-match" } });
    expect(screen.getByRole("status")).toHaveTextContent("No menu items match the current filters.");
  });

  it("quick-adds items without required modifiers", async () => {
    vi.stubGlobal("fetch", vi.fn().mockResolvedValue(jsonResponse(revampMenuResponse)));

    render(<CashierOrderScreen onSessionExpired={vi.fn()} />);

    fireEvent.click(await screen.findByRole("button", { name: /Muffin/ }));

    expect(screen.getByText("Muffin", { selector: ".cart-line__name" })).toBeVisible();
    expect(screen.getByText("Rp20.000", { selector: ".cart-line__price" })).toBeVisible();
  });

  it("cancels customization without changing existing cart lines", async () => {
    vi.stubGlobal("fetch", vi.fn().mockResolvedValue(jsonResponse(orderMenuResponse)));

    render(<CashierOrderScreen onSessionExpired={vi.fn()} />);

    await addAmericanoLine("Hot", "Normal");
    fireEvent.click(await screen.findByRole("button", { name: /Latte/ }));
    fireEvent.click(screen.getByLabelText("Iced +Rp2.000"));
    fireEvent.click(screen.getByLabelText("Less"));
    fireEvent.click(screen.getByRole("button", { name: "Cancel customization" }));

    expect(screen.getByText("Hot, Normal")).toBeVisible();
    expect(screen.queryByText("Iced, Less")).not.toBeInTheDocument();
    expect(screen.getByText("Americano", { selector: ".cart-line__name" })).toBeVisible();
  });
});

describe("CashierOrderScreen payment confirmation", () => {
  afterEach(() => {
    vi.restoreAllMocks();
    vi.unstubAllGlobals();
  });

  it("opens a confirm-payment dialog and returns focus to the trigger when backed out", async () => {
    vi.stubGlobal("fetch", vi.fn().mockResolvedValue(jsonResponse(orderMenuResponse)));

    render(<CashierOrderScreen onSessionExpired={vi.fn()} />);

    await addAmericanoLine("Hot", "Normal");
    fireEvent.click(screen.getByLabelText("QRIS"));

    const confirmTrigger = screen.getByRole("button", { name: "Proceed to Payment" });
    fireEvent.click(confirmTrigger);

    const dialog = screen.getByRole("dialog", { name: "Payment: QRIS" });
    expect(dialog).toBeVisible();
    expect(within(dialog).getByText("Rp19.980")).toBeVisible();
    expect(within(dialog).getByAltText("Static QRIS payment code")).toHaveAttribute("src", "/qris/static-qris.png");
    expect(within(dialog).getByText("Scan the QR code using your e-wallet or mobile banking.")).toBeVisible();
    expect(screen.getByRole("button", { name: "Cancel" })).toHaveFocus();

    fireEvent.click(screen.getByRole("button", { name: "Cancel" }));

    expect(screen.queryByRole("dialog", { name: "Payment: QRIS" })).not.toBeInTheDocument();
    expect(confirmTrigger).toHaveFocus();
    expect(screen.getByText("Hot, Normal")).toBeVisible();
    expect(screen.getByLabelText("QRIS")).toBeChecked();
  });

  it("opens a cash payment modal without QRIS image", async () => {
    vi.stubGlobal("fetch", vi.fn().mockResolvedValue(jsonResponse(orderMenuResponse)));

    render(<CashierOrderScreen onSessionExpired={vi.fn()} />);

    await addAmericanoLine("Hot", "Normal");
    fireEvent.click(screen.getByLabelText("Cash"));
    fireEvent.click(screen.getByRole("button", { name: "Proceed to Payment" }));

    const dialog = screen.getByRole("dialog", { name: "Payment: Cash" });
    expect(within(dialog).getByText("Rp19.980")).toBeVisible();
    expect(within(dialog).queryByAltText("Static QRIS payment code")).not.toBeInTheDocument();
  });

  it("creates a paid order with only backend-accepted request fields", async () => {
    const fetchMock = vi.fn((url: RequestInfo | URL, init?: RequestInit) => {
      if (String(url) === "/api/pos/orders" && init?.method === "POST") {
        return Promise.resolve(jsonResponse(paidOrderResponse, { status: 201 }));
      }

      return Promise.resolve(jsonResponse(revampMenuResponse));
    });
    vi.stubGlobal("fetch", fetchMock);
    vi.stubGlobal("crypto", { randomUUID: () => "11111111-1111-4111-8111-111111111111" });

    render(<CashierOrderScreen onSessionExpired={vi.fn()} />);

    await addAmericanoLine("Hot", "Normal");
    fireEvent.input(screen.getByLabelText("Order note"), { target: { value: "Less ice" } });
    fireEvent.click(screen.getByLabelText("Cash"));
    fireEvent.click(screen.getByRole("button", { name: "Proceed to Payment" }));
    fireEvent.click(within(screen.getByRole("dialog", { name: "Payment: Cash" })).getByRole("button", { name: "Confirm Paid" }));

    expect(await screen.findByText("Paid order created")).toBeVisible();
    expect(screen.getByText("Queue No. 001")).toBeVisible();

    const createCall = fetchMock.mock.calls.find(([url]) => String(url) === "/api/pos/orders");
    expect(JSON.parse(String(createCall?.[1]?.body))).toEqual({
        clientRequestId: "11111111-1111-4111-8111-111111111111",
        paymentMethod: "cash",
        note: "Less ice",
        lines: [
          {
            menuItemSlug: "americano",
            quantity: 1,
            modifiers: [
              { groupSlug: "temperature", optionSlug: "hot" },
              { groupSlug: "sugar", optionSlug: "normal" }
            ]
          }
        ]
      });
  });

  it("keeps the draft and reuses the same client request id when retrying after a recoverable error", async () => {
    const createBodies: string[] = [];
    const fetchMock = vi.fn((url: RequestInfo | URL, init?: RequestInit) => {
      if (String(url) === "/api/pos/orders" && init?.method === "POST") {
        createBodies.push(String(init.body));

        if (createBodies.length === 1) {
          return Promise.resolve(jsonResponse({ error: "invalid_order" }, { status: 422 }));
        }

        return Promise.resolve(jsonResponse(paidOrderResponse, { status: 200 }));
      }

      return Promise.resolve(jsonResponse(orderMenuResponse));
    });
    vi.stubGlobal("fetch", fetchMock);
    vi.stubGlobal("crypto", { randomUUID: () => "22222222-2222-4222-8222-222222222222" });

    render(<CashierOrderScreen onSessionExpired={vi.fn()} />);

    await addAmericanoLine("Hot", "Normal");
    fireEvent.click(screen.getByLabelText("Cash"));
    fireEvent.click(screen.getByRole("button", { name: "Proceed to Payment" }));
    fireEvent.click(within(screen.getByRole("dialog", { name: "Payment: Cash" })).getByRole("button", { name: "Confirm Paid" }));

    expect(await screen.findByRole("alert")).toHaveTextContent("The order is invalid. Check the draft and retry.");
    expect(screen.getByText("Hot, Normal")).toBeVisible();

    fireEvent.click(within(screen.getByRole("dialog", { name: "Payment: Cash" })).getByRole("button", { name: "Confirm Paid" }));

    expect(await screen.findByText("Paid order created")).toBeVisible();
    expect(createBodies).toHaveLength(2);
    expect(JSON.parse(createBodies[0]).clientRequestId).toBe("22222222-2222-4222-8222-222222222222");
    expect(JSON.parse(createBodies[1]).clientRequestId).toBe("22222222-2222-4222-8222-222222222222");
  });
});

describe("CashierOrderScreen paid order detail", () => {
  afterEach(() => {
    vi.restoreAllMocks();
    vi.unstubAllGlobals();
  });

  it("renders read-only paid detail and prints a minimal ticket", async () => {
    const printSpy = vi.spyOn(window, "print").mockImplementation(() => undefined);
    vi.stubGlobal(
      "fetch",
      vi.fn((url: RequestInfo | URL, init?: RequestInit) => {
        if (String(url) === "/api/pos/orders" && init?.method === "POST") {
          return Promise.resolve(jsonResponse(paidOrderResponse, { status: 201 }));
        }

        return Promise.resolve(jsonResponse(orderMenuResponse));
      })
    );
    vi.stubGlobal("crypto", { randomUUID: () => "33333333-3333-4333-8333-333333333333" });

    render(<CashierOrderScreen onSessionExpired={vi.fn()} />);

    await createPaidOrderInScreen();

    expect(screen.getByText("Queue No. 001")).toBeVisible();
    expect(screen.getByText("Status: Paid")).toBeVisible();
    expect(screen.getByText("Payment: Cash")).toBeVisible();
    expect(screen.getByText("Paid at: 2026-07-01T03:00:00Z")).toBeVisible();
    expect(screen.getByText("Americano")).toBeVisible();
    expect(screen.getByText("Hot, Normal")).toBeVisible();
    expect(screen.getByText("Note: Less ice")).toBeVisible();
    expect(screen.queryByLabelText("Order note")).not.toBeInTheDocument();

    fireEvent.click(screen.getByRole("button", { name: "Print Ticket" }));

    expect(screen.getByRole("region", { name: "Printable ticket" })).toBeVisible();
    expect(screen.getByText("Ticket Queue No. 001")).toBeVisible();
    expect(screen.getByText("Total Rp18.000")).toBeVisible();
    expect(printSpy).toHaveBeenCalledTimes(1);
  });

  it("starts a fresh unpaid draft from paid detail", async () => {
    vi.stubGlobal(
      "fetch",
      vi.fn((url: RequestInfo | URL, init?: RequestInit) => {
        if (String(url) === "/api/pos/orders" && init?.method === "POST") {
          return Promise.resolve(jsonResponse(paidOrderResponse, { status: 201 }));
        }

        return Promise.resolve(jsonResponse(orderMenuResponse));
      })
    );
    vi.stubGlobal("crypto", { randomUUID: () => "44444444-4444-4444-8444-444444444444" });

    render(<CashierOrderScreen onSessionExpired={vi.fn()} />);

    await createPaidOrderInScreen();
    fireEvent.click(screen.getByRole("button", { name: "Start New" }));

    expect(await screen.findByRole("heading", { level: 2, name: "New Order" })).toBeVisible();
    expect(screen.getByText("No items added yet.")).toBeVisible();
    expect(screen.queryByRole("button", { name: "Print Ticket" })).not.toBeInTheDocument();
  });

  it("cancels a paid order after confirmation and disables paid-only actions when cancelled", async () => {
    const cancelledOrder = {
      ...paidOrderResponse,
      status: "cancelled",
      cancelledAt: "2026-07-01T04:00:00Z"
    };
    vi.stubGlobal(
      "fetch",
      vi.fn((url: RequestInfo | URL, init?: RequestInit) => {
        if (String(url) === "/api/pos/orders" && init?.method === "POST") {
          return Promise.resolve(jsonResponse(paidOrderResponse, { status: 201 }));
        }

        if (String(url) === "/api/pos/orders/order-1/cancel" && init?.method === "POST") {
          return Promise.resolve(jsonResponse(cancelledOrder));
        }

        return Promise.resolve(jsonResponse(orderMenuResponse));
      })
    );
    vi.stubGlobal("crypto", { randomUUID: () => "55555555-5555-4555-8555-555555555555" });

    render(<CashierOrderScreen onSessionExpired={vi.fn()} />);

    await createPaidOrderInScreen();
    fireEvent.click(screen.getByRole("button", { name: "Cancel Order" }));

    expect(screen.getByRole("dialog", { name: "Cancel order" })).toBeVisible();
    fireEvent.click(within(screen.getByRole("dialog", { name: "Cancel order" })).getByRole("button", { name: "Back" }));
    expect(screen.queryByRole("dialog", { name: "Cancel order" })).not.toBeInTheDocument();

    fireEvent.click(screen.getByRole("button", { name: "Cancel Order" }));
    fireEvent.click(within(screen.getByRole("dialog", { name: "Cancel order" })).getByRole("button", { name: "Cancel Order" }));

    expect(await screen.findByText("Status: Cancelled")).toBeVisible();
    expect(screen.getByText("Cancelled at: 2026-07-01T04:00:00Z")).toBeVisible();
    expect(screen.queryByRole("button", { name: "Cancel Order" })).not.toBeInTheDocument();
    expect(screen.getByRole("button", { name: "Print Ticket" })).toBeDisabled();
  });

  it("shows non-destructive cancellation errors", async () => {
    vi.stubGlobal(
      "fetch",
      vi.fn((url: RequestInfo | URL, init?: RequestInit) => {
        if (String(url) === "/api/pos/orders" && init?.method === "POST") {
          return Promise.resolve(jsonResponse(paidOrderResponse, { status: 201 }));
        }

        if (String(url) === "/api/pos/orders/order-1/cancel" && init?.method === "POST") {
          return Promise.resolve(jsonResponse({ error: "order_not_cancellable" }, { status: 409 }));
        }

        return Promise.resolve(jsonResponse(orderMenuResponse));
      })
    );
    vi.stubGlobal("crypto", { randomUUID: () => "66666666-6666-4666-8666-666666666666" });

    render(<CashierOrderScreen onSessionExpired={vi.fn()} />);

    await createPaidOrderInScreen();
    fireEvent.click(screen.getByRole("button", { name: "Cancel Order" }));
    fireEvent.click(within(screen.getByRole("dialog", { name: "Cancel order" })).getByRole("button", { name: "Cancel Order" }));

    expect(await screen.findByRole("alert")).toHaveTextContent(
      "This order can no longer be cancelled from this screen."
    );
    expect(screen.getByText("Status: Paid")).toBeVisible();
  });
});

async function addAmericanoLine(temperatureLabel: string, sugarLabel: string) {
  fireEvent.click(await screen.findByRole("button", { name: /^Americano Rp18\.000$/ }));
  fireEvent.click(screen.getByLabelText(temperatureLabel));
  fireEvent.click(screen.getByLabelText(sugarLabel));
  fireEvent.click(screen.getByRole("button", { name: "Add Item To Order" }));
}

async function createPaidOrderInScreen() {
  await addAmericanoLine("Hot", "Normal");
  fireEvent.input(screen.getByLabelText("Order note"), { target: { value: "Less ice" } });
  fireEvent.click(screen.getByLabelText("Cash"));
  fireEvent.click(screen.getByRole("button", { name: "Proceed to Payment" }));
  fireEvent.click(within(screen.getByRole("dialog", { name: "Payment: Cash" })).getByRole("button", { name: "Confirm Paid" }));
  await screen.findByText("Paid order created");
}

function jsonResponse(body: unknown, init: ResponseInit = {}) {
  return new Response(JSON.stringify(body), {
    headers: { "Content-Type": "application/json" },
    status: 200,
    ...init
  });
}
