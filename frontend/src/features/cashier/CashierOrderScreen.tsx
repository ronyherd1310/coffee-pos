import { useEffect, useRef, useState } from "preact/hooks";
import { formatRupiah } from "../../lib/format";
import { cancelPaidOrder, createPaidOrder, getCashierMenu } from "../../lib/pos";
import { CancelOrderDialog } from "./CancelOrderDialog";
import {
  buildCatalogCategories,
  buildCatalogItems,
  type CatalogSort,
  type QuickFilter
} from "./catalogView";
import { ConfirmPaymentDialog } from "./ConfirmPaymentDialog";
import { PaidOrderDetail as PaidOrderDetailView } from "./PaidOrderDetail";
import {
  buildCreatePaidOrderPayload,
  calculateDraftBreakdown,
  calculateLineTotal,
  clampQuantity,
  createCartLine,
  validateDraft,
  type CartLine,
  type SelectedModifiers
} from "./orderDraft";
import type { CashierMenu, MenuItem, PaidOrderDetail, PaymentMethod } from "./types";

type CashierOrderScreenProps = {
  onSessionExpired: () => void;
};

type MenuState =
  | { status: "loading" }
  | { status: "loaded"; menu: CashierMenu }
  | { status: "empty" }
  | { status: "error"; message: string };

const quickFilterOptions: { label: string; value: QuickFilter }[] = [
  { label: "🔥 Best Seller", value: "bestSeller" },
  { label: "❄️ Iced", value: "iced" },
  { label: "◇ Low Sugar", value: "lowSugar" },
  { label: "✨ New Arrival", value: "newArrival" }
];

export function CashierOrderScreen({ onSessionExpired }: CashierOrderScreenProps) {
  const [menuState, setMenuState] = useState<MenuState>({ status: "loading" });
  const [selectedItem, setSelectedItem] = useState<MenuItem | undefined>();
  const [selectedModifiers, setSelectedModifiers] = useState<SelectedModifiers>({});
  const [selectedQuantity, setSelectedQuantity] = useState(1);
  const [searchQuery, setSearchQuery] = useState("");
  const [activeCategorySlug, setActiveCategorySlug] = useState("all");
  const [quickFilters, setQuickFilters] = useState<QuickFilter[]>([]);
  const [catalogSort, setCatalogSort] = useState<CatalogSort>("popular");
  const [cartLines, setCartLines] = useState<CartLine[]>([]);
  const [note, setNote] = useState("");
  const [paymentMethod, setPaymentMethod] = useState<PaymentMethod | undefined>();
  const [isConfirmDialogOpen, setIsConfirmDialogOpen] = useState(false);
  const [isSubmittingPayment, setIsSubmittingPayment] = useState(false);
  const [pendingClientRequestId, setPendingClientRequestId] = useState<string | undefined>();
  const [paymentError, setPaymentError] = useState<string | undefined>();
  const [paidOrder, setPaidOrder] = useState<PaidOrderDetail | undefined>();
  const [showPrintableTicket, setShowPrintableTicket] = useState(false);
  const [isCancelDialogOpen, setIsCancelDialogOpen] = useState(false);
  const [isCancellingOrder, setIsCancellingOrder] = useState(false);
  const [cancelError, setCancelError] = useState<string | undefined>();
  const confirmButtonRef = useRef<HTMLButtonElement>(null);

  async function loadMenu() {
    setMenuState({ status: "loading" });

    const result = await getCashierMenu();

    if (result.status === "unauthorized") {
      onSessionExpired();
      return;
    }

    if (result.status === "success") {
      const itemCount = result.menu.categories.reduce(
        (total, category) => total + category.items.length,
        0
      );
      setMenuState(itemCount > 0 ? { menu: result.menu, status: "loaded" } : { status: "empty" });
      return;
    }

    setMenuState({ message: "Cannot load the cashier menu.", status: "error" });
  }

  useEffect(() => {
    void loadMenu();
  }, []);

  if (menuState.status === "loading") {
    return (
      <section className="cashier-screen cashier-screen--state" aria-labelledby="cashier-title">
        <h2 id="cashier-title">New Order</h2>
        <p className="status-message" role="status">
          Loading menu...
        </p>
      </section>
    );
  }

  if (menuState.status === "error") {
    return (
      <section className="cashier-screen cashier-screen--state" aria-labelledby="cashier-title">
        <h2 id="cashier-title">New Order</h2>
        <p className="cashier-alert" role="alert">
          {menuState.message}
        </p>
        <button className="button button--secondary" type="button" onClick={() => void loadMenu()}>
          Retry menu
        </button>
      </section>
    );
  }

  if (menuState.status === "empty") {
    return (
      <section className="cashier-screen cashier-screen--state" aria-labelledby="cashier-title">
        <h2 id="cashier-title">New Order</h2>
        <p className="status-message" role="status">
          No menu items available.
        </p>
      </section>
    );
  }

  if (paidOrder) {
    return (
      <section className="cashier-screen paid-order-screen" aria-labelledby="paid-order-title">
        <PaidOrderDetailView
          cancelError={cancelError}
          order={paidOrder}
          showPrintableTicket={showPrintableTicket}
          onCancelClick={() => {
            setCancelError(undefined);
            setIsCancelDialogOpen(true);
          }}
          onPrintClick={() => {
            setShowPrintableTicket(true);
            window.print();
          }}
          onStartNew={() => {
            setPaidOrder(undefined);
            setShowPrintableTicket(false);
            setCancelError(undefined);
            setIsCancelDialogOpen(false);
          }}
        />

        {isCancelDialogOpen ? (
          <CancelOrderDialog
            isSubmitting={isCancellingOrder}
            onBack={() => setIsCancelDialogOpen(false)}
            onConfirm={() => void handleCancelPaidOrder(paidOrder)}
          />
        ) : null}
      </section>
    );
  }

  const draft = { lines: cartLines, note, paymentMethod };
  const validation = validateDraft(draft);
  const breakdown = calculateDraftBreakdown(cartLines);
  const totalRp = breakdown.totalRp;
  const catalogCategories = buildCatalogCategories(menuState.menu);
  const catalogItems = buildCatalogItems(menuState.menu, {
    categorySlug: activeCategorySlug,
    quickFilters,
    searchQuery,
    sort: catalogSort
  });

  function addCartLine(item: MenuItem, quantity: number, modifiers: SelectedModifiers) {
    setCartLines((current) => [
      ...current,
      createCartLine({
        id: newCartLineId(),
        item,
        quantity,
        selectedModifiers: modifiers
      })
    ]);
  }

  async function handleConfirmPaidSubmit() {
    if (!paymentMethod || validation.isValid === false) {
      return;
    }

    const clientRequestId = pendingClientRequestId ?? newClientRequestId();
    setPendingClientRequestId(clientRequestId);
    setIsSubmittingPayment(true);
    setPaymentError(undefined);

    const result = await createPaidOrder(
      buildCreatePaidOrderPayload({
        clientRequestId,
        draft
      })
    );

    setIsSubmittingPayment(false);

    if (result.status === "success") {
      setPaidOrder(result.order);
      setCartLines([]);
      setNote("");
      setPaymentMethod(undefined);
      setSelectedItem(undefined);
      setSelectedModifiers({});
      setSelectedQuantity(1);
      setPendingClientRequestId(undefined);
      setIsConfirmDialogOpen(false);
      return;
    }

    if (result.status === "unauthorized") {
      onSessionExpired();
      return;
    }

    setPaymentError(createOrderErrorMessage(result.status));
  }

  async function handleCancelPaidOrder(order: PaidOrderDetail) {
    setIsCancellingOrder(true);
    setCancelError(undefined);

    const result = await cancelPaidOrder(order.orderId);

    setIsCancellingOrder(false);

    if (result.status === "success") {
      setPaidOrder(result.order);
      setShowPrintableTicket(false);
      setIsCancelDialogOpen(false);
      return;
    }

    if (result.status === "unauthorized") {
      onSessionExpired();
      return;
    }

    setCancelError(cancelOrderErrorMessage(result.status));
    setIsCancelDialogOpen(false);
  }

  return (
    <section className="cashier-screen" aria-labelledby="cashier-title">
      <div className="cashier-screen__header">
        <div>
          <p className="cashier-screen__eyebrow">Cashier order entry</p>
          <h2 id="cashier-title">New Order</h2>
        </div>
      </div>

      <label className="catalog-search">
        <span className="sr-only">Search menu item</span>
        <input
          onInput={(event) => setSearchQuery((event.currentTarget as HTMLInputElement).value)}
          placeholder="Search menu item..."
          type="search"
          value={searchQuery}
        />
      </label>

      <div className="cashier-layout">
        <aside className="cashier-sidebar">
          <section className="cashier-panel menu-panel" aria-labelledby="menu-title">
            <div className="catalog-toolbar">
              <div>
                <h3 id="menu-title">Menu</h3>
                <p>Choose an item from the catalog.</p>
              </div>
            </div>

            <div className="catalog-tabs" role="tablist" aria-label="Menu categories">
              {catalogCategories.map((category) => (
                <button
                  aria-selected={activeCategorySlug === category.slug}
                  className={activeCategorySlug === category.slug ? "catalog-tab catalog-tab--active" : "catalog-tab"}
                  key={category.slug}
                  onClick={() => setActiveCategorySlug(category.slug)}
                  role="tab"
                  type="button"
                >
                  {category.name}
                </button>
              ))}
            </div>

            <div className="catalog-filters" aria-label="Quick filters">
              <span className="catalog-filters__label">Quick Filters:</span>
              {quickFilterOptions.map((filter) => (
                <button
                  aria-label={filter.value === "bestSeller" ? "Best Seller" : filter.label.replace(/^[^A-Za-z]+ /, "")}
                  aria-pressed={quickFilters.includes(filter.value)}
                  className={
                    quickFilters.includes(filter.value)
                      ? "catalog-filter catalog-filter--active"
                      : "catalog-filter"
                  }
                  key={filter.value}
                  onClick={() => {
                    setQuickFilters((current) =>
                      current.includes(filter.value)
                        ? current.filter((value) => value !== filter.value)
                        : [...current, filter.value]
                    );
                  }}
                  type="button"
                >
                  {filter.label}
                </button>
              ))}

              <label className="catalog-sort">
                <span>Sort by:</span>
                <select
                  aria-label="Sort menu"
                  onChange={(event) => setCatalogSort((event.currentTarget as HTMLSelectElement).value as CatalogSort)}
                  value={catalogSort}
                >
                  <option value="popular">Popular</option>
                </select>
              </label>
            </div>

            {catalogItems.length === 0 ? (
              <p className="catalog-empty" role="status">
                No menu items match the current filters.
              </p>
            ) : (
              <div className="menu-list menu-list--grid">
                {catalogItems.map(({ item }) => (
                  <button
                    aria-pressed={selectedItem?.slug === item.slug}
                    className="menu-item menu-card"
                    key={item.slug}
                    onClick={() => {
                      if (!hasRequiredModifierGroups(item)) {
                        addCartLine(item, 1, {});
                        setSelectedItem(undefined);
                        setSelectedModifiers({});
                        setSelectedQuantity(1);
                        return;
                      }

                      setSelectedItem(item);
                      setSelectedModifiers({});
                      setSelectedQuantity(1);
                    }}
                    type="button"
                  >
                    <span className="menu-item__thumb" aria-hidden="true">
                      <img alt="" src={menuItemImageSrc(item)} />
                    </span>
                    <span className="menu-item__content">
                      {menuItemBadges(item).length > 0 ? (
                        <span className="menu-card__badges" aria-hidden="true">
                          {menuItemBadges(item).map((badge) => (
                            <span className="menu-card__badge" key={badge}>
                              {badge}
                            </span>
                          ))}
                        </span>
                      ) : null}
                      <span>{item.name}</span>
                      <span>{formatRupiah(item.priceRp)}</span>
                    </span>
                  </button>
                ))}
              </div>
            )}
          </section>

          {selectedItem ? <h3 className="cashier-sidebar__section-title">Selected item</h3> : null}

          {selectedItem ? (
            <SelectedItemPanel
              item={selectedItem}
              quantity={selectedQuantity}
              selectedModifiers={selectedModifiers}
              onModifierChange={(groupSlug, optionSlug) =>
                setSelectedModifiers((current) => ({ ...current, [groupSlug]: optionSlug }))
              }
              onQuantityChange={(nextQuantity) => setSelectedQuantity(clampQuantity(nextQuantity))}
              onCancel={() => {
                setSelectedItem(undefined);
                setSelectedModifiers({});
                setSelectedQuantity(1);
              }}
              onAddLine={() => {
                if (!selectedItem || !hasRequiredModifiers(selectedItem, selectedModifiers)) {
                  return;
                }

                addCartLine(selectedItem, selectedQuantity, selectedModifiers);
                setSelectedModifiers({});
                setSelectedQuantity(1);
              }}
            />
          ) : null}
        </aside>

        <CurrentOrderPanel
          canConfirm={validation.isValid}
          confirmButtonRef={confirmButtonRef}
          lines={cartLines}
          note={note}
          paymentMethod={paymentMethod}
          subtotalRp={breakdown.subtotalRp}
          taxRp={breakdown.taxRp}
          totalRp={breakdown.totalRp}
          onNoteChange={setNote}
          onConfirmClick={() => {
            setPaymentError(undefined);
            setIsConfirmDialogOpen(true);
          }}
          onPaymentMethodChange={setPaymentMethod}
          onQuantityChange={(lineId, quantity) =>
            setCartLines((current) =>
              current.map((line) =>
                line.id === lineId ? { ...line, quantity: clampQuantity(quantity) } : line
              )
            )
          }
          onRemoveLine={(lineId) => setCartLines((current) => current.filter((line) => line.id !== lineId))}
        />

      </div>

      {isConfirmDialogOpen ? (
        <ConfirmPaymentDialog
          error={paymentError}
          isSubmitting={isSubmittingPayment}
          paymentMethod={paymentMethod}
          totalRp={totalRp}
          onBack={() => {
            setIsConfirmDialogOpen(false);
            setPaymentError(undefined);
            confirmButtonRef.current?.focus();
          }}
          onConfirm={() => void handleConfirmPaidSubmit()}
        />
      ) : null}
    </section>
  );
}

type SelectedItemPanelProps = {
  item: MenuItem | undefined;
  quantity: number;
  selectedModifiers: SelectedModifiers;
  onModifierChange: (groupSlug: string, optionSlug: string) => void;
  onQuantityChange: (quantity: number) => void;
  onCancel: () => void;
  onAddLine: () => void;
};

function SelectedItemPanel({
  item,
  quantity,
  selectedModifiers,
  onModifierChange,
  onQuantityChange,
  onCancel,
  onAddLine
}: SelectedItemPanelProps) {
  if (!item) {
    return (
      <section className="cashier-panel selected-item-panel" aria-labelledby="selected-item-title">
        <h3 id="selected-item-title">Select an item</h3>
        <p>Choose a menu item to set modifiers and quantity.</p>
      </section>
    );
  }

  const canAdd = hasRequiredModifiers(item, selectedModifiers);

  return (
    <section className="cashier-panel selected-item-panel" aria-labelledby="selected-item-title">
      <div className="selected-item-panel__title">
        <span className="menu-item__thumb menu-item__thumb--small" aria-hidden="true">
          <img alt="" src={menuItemImageSrc(item)} />
        </span>
        <div>
          <h3 id="selected-item-title" aria-label={`Configure ${item.name}`}>
            {item.name}
          </h3>
          <p>{formatRupiah(item.priceRp)}</p>
        </div>
      </div>

      {item.modifierGroups.map((group) => (
        <fieldset className="modifier-group" key={group.slug}>
          <legend>
            {group.name}
            {group.required ? <span>Required</span> : null}
          </legend>
          <div className="option-grid">
            {group.options.map((option) => {
              const label =
                option.priceDeltaRp > 0
                  ? `${option.name} +${formatRupiah(option.priceDeltaRp)}`
                  : option.name;
              const id = `modifier-${item.slug}-${group.slug}-${option.slug}`;

              return (
                <label className={optionControlClass(option.slug)} htmlFor={id} key={option.slug}>
                  <input
                    checked={selectedModifiers[group.slug] === option.slug}
                    id={id}
                    name={`modifier-${item.slug}-${group.slug}`}
                    onChange={() => onModifierChange(group.slug, option.slug)}
                    type="radio"
                  />
                  <span className="option-control__label">{label}</span>
                </label>
              );
            })}
          </div>
        </fieldset>
      ))}

      <div className="quantity-row">
        <span>Quantity</span>
        <div className="quantity-stepper">
          <button
            aria-label="Decrease selected item quantity"
            className="stepper-button"
            onClick={() => onQuantityChange(quantity - 1)}
            type="button"
          >
            -
          </button>
          <input aria-label="Selected item quantity" readOnly type="number" value={quantity} />
          <button
            aria-label="Increase selected item quantity"
            className="stepper-button"
            onClick={() => onQuantityChange(quantity + 1)}
            type="button"
          >
            +
          </button>
        </div>
      </div>

      <div className="selected-item-panel__actions">
        <button className="button button--secondary" onClick={onCancel} type="button">
          <span>Cancel customization</span>
        </button>
        <button className="button button--primary button--add-item" disabled={!canAdd} onClick={onAddLine} type="button">
          <span>Add Item To Order</span>
        </button>
      </div>
    </section>
  );
}

type CurrentOrderPanelProps = {
  canConfirm: boolean;
  confirmButtonRef: { current: HTMLButtonElement | null };
  lines: CartLine[];
  note: string;
  paymentMethod: PaymentMethod | undefined;
  subtotalRp: number;
  taxRp: number;
  totalRp: number;
  onConfirmClick: () => void;
  onNoteChange: (note: string) => void;
  onPaymentMethodChange: (method: PaymentMethod) => void;
  onQuantityChange: (lineId: string, quantity: number) => void;
  onRemoveLine: (lineId: string) => void;
};

function CurrentOrderPanel({
  canConfirm,
  confirmButtonRef,
  lines,
  note,
  paymentMethod,
  subtotalRp,
  taxRp,
  totalRp,
  onConfirmClick,
  onNoteChange,
  onPaymentMethodChange,
  onQuantityChange,
  onRemoveLine
}: CurrentOrderPanelProps) {
  return (
    <section className="cashier-panel current-order-panel" aria-labelledby="current-order-title">
      <div className="current-order-panel__header">
        <h3 id="current-order-title">Current Order</h3>
        <p>#ORD-0142</p>
      </div>

      {lines.length === 0 ? (
        <p>No items added yet.</p>
      ) : (
        <ul className="cart-lines" aria-label="Current order items">
          {lines.map((line) => (
            <li className="cart-line" key={line.id}>
              <div>
                <p className="cart-line__quantity">{line.quantity}x</p>
              </div>
              <span className="cart-line__thumb" aria-hidden="true">
                <img alt="" src={menuItemImageSrc(line.item)} />
              </span>
              <div className="cart-line__main">
                <p className="cart-line__name">{line.item.name}</p>
                <p>{formatModifierSummary(line)}</p>
              </div>
              <p className="cart-line__price">{formatRupiah(calculateLineTotal(line))}</p>
              <div className="cart-line__actions">
                <div className="quantity-stepper">
                  <button
                    aria-label={`Decrease ${line.item.name} quantity`}
                    className="stepper-button"
                    onClick={() => onQuantityChange(line.id, line.quantity - 1)}
                    type="button"
                  >
                    -
                  </button>
                  <span aria-label={`${line.item.name} quantity`}>{line.quantity}</span>
                  <button
                    aria-label={`Increase ${line.item.name} quantity`}
                    className="stepper-button"
                    onClick={() => onQuantityChange(line.id, line.quantity + 1)}
                    type="button"
                  >
                    +
                  </button>
                </div>
                <button
                  aria-label={`Remove ${line.item.name}`}
                  className="button button--secondary button--remove-line"
                  onClick={() => onRemoveLine(line.id)}
                  type="button"
                >
                  <span>Remove</span>
                </button>
              </div>
            </li>
          ))}
        </ul>
      )}

      <label className="note-field">
        <span>Order note</span>
        <textarea
          placeholder="Add a note for this order..."
          maxLength={120}
          onInput={(event) => onNoteChange((event.currentTarget as HTMLTextAreaElement).value.slice(0, 120))}
          rows={3}
          value={note}
        />
      </label>
      <p className="note-count">{note.length} / 120</p>

      <dl className="order-total">
        <div>
          <dt>Subtotal</dt>
          <dd>{formatRupiah(subtotalRp)}</dd>
        </div>
        <div className="order-total__tax">
          <dt>Tax</dt>
          <dd>{formatRupiah(taxRp)}</dd>
        </div>
        <div>
          <dt>Total</dt>
          <dd className="payment-total">{formatRupiah(totalRp)}</dd>
        </div>
      </dl>

      <fieldset className="payment-methods order-payment-methods">
        <legend>Payment Method</legend>
        <label className="option-control option-control--cash" htmlFor="payment-cash">
          <input
            checked={paymentMethod === "cash"}
            id="payment-cash"
            name="payment-method"
            onChange={() => onPaymentMethodChange("cash")}
            type="radio"
          />
          <span className="option-control__label">Cash</span>
        </label>
        <label className="option-control option-control--qris" htmlFor="payment-qris">
          <input
            checked={paymentMethod === "qris"}
            id="payment-qris"
            name="payment-method"
            onChange={() => onPaymentMethodChange("qris")}
            type="radio"
          />
          <span className="option-control__label">QRIS</span>
        </label>
      </fieldset>

      <button
        className="button button--primary button--confirm-paid"
        disabled={!canConfirm}
        onClick={onConfirmClick}
        ref={confirmButtonRef}
        type="button"
      >
        <span>Proceed to Payment</span>
      </button>
    </section>
  );
}

function hasRequiredModifiers(item: MenuItem, selectedModifiers: SelectedModifiers): boolean {
  return item.modifierGroups.every((group) => !group.required || Boolean(selectedModifiers[group.slug]));
}

function hasRequiredModifierGroups(item: MenuItem): boolean {
  return item.modifierGroups.some((group) => group.required);
}

function menuItemImageSrc(item: MenuItem): string {
  if (item.imagePath) {
    return item.imagePath;
  }

  const fallbackBySlug: Record<string, string> = {
    americano: "/menu/americano.png",
    latte: "/menu/latte.png"
  };

  return fallbackBySlug[item.slug] ?? "/menu/americano.png";
}

function menuItemBadges(item: MenuItem): string[] {
  if (item.bestSeller) {
    return ["Best Seller"];
  }
  if (item.newArrival) {
    return ["New Arrival"];
  }
  if (item.promo) {
    return ["Promo"];
  }
  return [];
}

function optionControlClass(optionSlug: string): string {
  return `option-control option-control--${optionSlug}`;
}

function formatModifierSummary(line: CartLine): string {
  return line.item.modifierGroups
    .map((group) => {
      const optionSlug = line.selectedModifiers[group.slug];
      return group.options.find((option) => option.slug === optionSlug)?.name;
    })
    .filter((name): name is string => Boolean(name))
    .join(", ");
}

function newCartLineId(): string {
  return globalThis.crypto?.randomUUID?.() ?? `${Date.now()}-${Math.random()}`;
}

function newClientRequestId(): string {
  return globalThis.crypto.randomUUID().toLowerCase();
}

function createOrderErrorMessage(status: Exclude<Awaited<ReturnType<typeof createPaidOrder>>["status"], "success">): string {
  switch (status) {
    case "invalid-order":
      return "The order is invalid. Check the draft and retry.";
    case "idempotency-conflict":
      return "This payment confirmation conflicts with an earlier request. Start a new order and try again.";
    case "invalid-client-request-id":
    case "unexpected":
      return "Cannot confirm this payment right now. Start a new order and try again.";
    case "unavailable":
      return "Cannot reach the order service. Check the connection and retry.";
    case "unauthorized":
      return "Your session expired. Sign in again to continue.";
  }
}

function cancelOrderErrorMessage(status: Exclude<Awaited<ReturnType<typeof cancelPaidOrder>>["status"], "success">): string {
  switch (status) {
    case "not-cancellable":
      return "This order can no longer be cancelled from this screen.";
    case "not-found":
      return "This order could not be found. It was not cancelled.";
    case "unavailable":
      return "Cannot reach the order service. Check the connection and retry.";
    case "unexpected":
      return "Cannot cancel this order right now. Try again later.";
    case "unauthorized":
      return "Your session expired. Sign in again to continue.";
  }
}
