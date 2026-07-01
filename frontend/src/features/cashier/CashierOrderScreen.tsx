import { useEffect, useRef, useState } from "preact/hooks";
import { cancelPaidOrder, createPaidOrder, getCashierMenu } from "../../lib/pos";
import { CancelOrderDialog } from "./CancelOrderDialog";
import {
  buildCatalogCategories,
  buildCatalogItems,
  type CatalogSort,
  type QuickFilter
} from "./catalogView";
import { hasRequiredModifierGroups, hasRequiredModifiers } from "./cashierItemView";
import { ConfirmPaymentDialog } from "./ConfirmPaymentDialog";
import { CurrentOrderPanel } from "./CurrentOrderPanel";
import { MenuCatalogPanel } from "./MenuCatalogPanel";
import { PaidOrderDetail as PaidOrderDetailView } from "./PaidOrderDetail";
import {
  buildCreatePaidOrderPayload,
  calculateDraftBreakdown,
  clampQuantity,
  createCartLine,
  validateDraft,
  type CartLine,
  type SelectedModifiers
} from "./orderDraft";
import { SelectedItemPanel } from "./SelectedItemPanel";
import type { CashierMenu, MenuItem, PaidOrderDetail, PaymentMethod } from "./types";

type CashierOrderScreenProps = {
  onSessionExpired: () => void;
};

type MenuState =
  | { status: "loading" }
  | { status: "loaded"; menu: CashierMenu }
  | { status: "empty" }
  | { status: "error"; message: string };

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
          onStartNew={resetPaidOrderState}
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

  function resetSelectedItem() {
    setSelectedItem(undefined);
    setSelectedModifiers({});
    setSelectedQuantity(1);
  }

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

  function handleCatalogItemClick(item: MenuItem) {
    if (!hasRequiredModifierGroups(item)) {
      addCartLine(item, 1, {});
      resetSelectedItem();
      return;
    }

    setSelectedItem(item);
    setSelectedModifiers({});
    setSelectedQuantity(1);
  }

  function handleQuickFilterToggle(filter: QuickFilter) {
    setQuickFilters((current) =>
      current.includes(filter) ? current.filter((value) => value !== filter) : [...current, filter]
    );
  }

  function handleSelectedItemAddLine() {
    if (!selectedItem || !hasRequiredModifiers(selectedItem, selectedModifiers)) {
      return;
    }

    addCartLine(selectedItem, selectedQuantity, selectedModifiers);
    setSelectedModifiers({});
    setSelectedQuantity(1);
  }

  function resetPaidOrderState() {
    setPaidOrder(undefined);
    setShowPrintableTicket(false);
    setCancelError(undefined);
    setIsCancelDialogOpen(false);
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
          <MenuCatalogPanel
            activeCategorySlug={activeCategorySlug}
            catalogCategories={catalogCategories}
            catalogItems={catalogItems}
            catalogSort={catalogSort}
            quickFilters={quickFilters}
            selectedItemSlug={selectedItem?.slug}
            onCategoryChange={setActiveCategorySlug}
            onItemClick={handleCatalogItemClick}
            onQuickFilterToggle={handleQuickFilterToggle}
            onSortChange={setCatalogSort}
          />

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
              onCancel={resetSelectedItem}
              onAddLine={handleSelectedItemAddLine}
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
