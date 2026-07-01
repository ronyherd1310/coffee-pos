import { useEffect, useRef, useState } from "preact/hooks";
import { formatRupiah } from "../../lib/format";
import { cancelPaidOrder, createPaidOrder, getCashierMenu } from "../../lib/pos";
import { CancelOrderDialog } from "./CancelOrderDialog";
import { ConfirmPaymentDialog } from "./ConfirmPaymentDialog";
import { PaidOrderDetail as PaidOrderDetailView } from "./PaidOrderDetail";
import {
  buildCreatePaidOrderPayload,
  calculateDraftTotal,
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

export function CashierOrderScreen({ onSessionExpired }: CashierOrderScreenProps) {
  const [menuState, setMenuState] = useState<MenuState>({ status: "loading" });
  const [selectedItem, setSelectedItem] = useState<MenuItem | undefined>();
  const [selectedModifiers, setSelectedModifiers] = useState<SelectedModifiers>({});
  const [selectedQuantity, setSelectedQuantity] = useState(1);
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
  const totalRp = calculateDraftTotal(cartLines);

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

      <div className="cashier-layout">
        <aside className="cashier-sidebar">
          <section className="cashier-panel menu-panel" aria-labelledby="menu-title">
            <h3 id="menu-title">Menu</h3>
            {menuState.menu.categories.map((category) => (
              <div className="menu-category" key={category.slug}>
                <h4>{category.name}</h4>
                <div className="menu-list">
                  {category.items.map((item) => (
                    <button
                      aria-pressed={selectedItem?.slug === item.slug}
                      className="menu-item"
                      key={item.slug}
                      onClick={() => {
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
                        <span>{item.name}</span>
                        <span>{formatRupiah(item.priceRp)}</span>
                      </span>
                    </button>
                  ))}
                </div>
              </div>
            ))}
          </section>

          <h3 className="cashier-sidebar__section-title">Selected item</h3>

          <SelectedItemPanel
            item={selectedItem}
            quantity={selectedQuantity}
            selectedModifiers={selectedModifiers}
            onModifierChange={(groupSlug, optionSlug) =>
              setSelectedModifiers((current) => ({ ...current, [groupSlug]: optionSlug }))
            }
            onQuantityChange={(nextQuantity) => setSelectedQuantity(clampQuantity(nextQuantity))}
            onAddLine={() => {
              if (!selectedItem || !hasRequiredModifiers(selectedItem, selectedModifiers)) {
                return;
              }

              setCartLines((current) => [
                ...current,
                createCartLine({
                  id: newCartLineId(),
                  item: selectedItem,
                  quantity: selectedQuantity,
                  selectedModifiers
                })
              ]);
              setSelectedModifiers({});
              setSelectedQuantity(1);
            }}
          />
        </aside>

        <CurrentOrderPanel
          canConfirm={validation.isValid}
          confirmButtonRef={confirmButtonRef}
          lines={cartLines}
          note={note}
          paymentMethod={paymentMethod}
          totalRp={totalRp}
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

        <PaymentPreviewPanel paymentMethod={paymentMethod} />
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
  onAddLine: () => void;
};

function SelectedItemPanel({
  item,
  quantity,
  selectedModifiers,
  onModifierChange,
  onQuantityChange,
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

      <button className="button button--primary button--add-item" disabled={!canAdd} onClick={onAddLine} type="button">
        <span>Add Item To Order</span>
      </button>
    </section>
  );
}

type CurrentOrderPanelProps = {
  canConfirm: boolean;
  confirmButtonRef: { current: HTMLButtonElement | null };
  lines: CartLine[];
  note: string;
  paymentMethod: PaymentMethod | undefined;
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
  totalRp,
  onConfirmClick,
  onNoteChange,
  onPaymentMethodChange,
  onQuantityChange,
  onRemoveLine
}: CurrentOrderPanelProps) {
  return (
    <section className="cashier-panel current-order-panel" aria-labelledby="current-order-title">
      <h3 id="current-order-title">Current Order</h3>

      {lines.length === 0 ? (
        <p>No items added yet.</p>
      ) : (
        <ul className="cart-lines" aria-label="Current order items">
          {lines.map((line) => (
            <li className="cart-line" key={line.id}>
              <div>
                <p className="cart-line__quantity">{line.quantity}x</p>
              </div>
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
                  className="button button--secondary button--compact button--remove-line"
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

      <div className="order-actions">
        <button
          className="button button--primary button--confirm-paid"
          disabled={!canConfirm}
          onClick={onConfirmClick}
          ref={confirmButtonRef}
          type="button"
        >
          <span>Confirm Paid</span>
        </button>
        <button aria-label="Print Ticket" className="button button--secondary button--print-ticket" disabled type="button">
          <span>
            <span>Print Ticket</span>
            <small aria-hidden="true">(Disabled)</small>
          </span>
        </button>
        <p>
          Status: <span>Not paid</span>
        </p>
      </div>

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
          <dd>{formatRupiah(totalRp)}</dd>
        </div>
        <div>
          <dt>Total</dt>
          <dd className="payment-total">{formatRupiah(totalRp)}</dd>
        </div>
        <div className="order-total__payment">
          <dt>Payment method <span>(required)</span></dt>
          <dd>
            <fieldset className="payment-methods">
              <legend>Payment method</legend>
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
          </dd>
        </div>
      </dl>
    </section>
  );
}

type PaymentPreviewPanelProps = {
  paymentMethod: PaymentMethod | undefined;
};

function PaymentPreviewPanel({ paymentMethod }: PaymentPreviewPanelProps) {
  return (
    <section className="cashier-panel payment-panel" aria-labelledby="payment-title">
      <h3 id="payment-title">Payment</h3>
      <p className="payment-preview-title">
        Payment: {paymentMethod ? paymentMethod.toUpperCase() : "Select method"}
      </p>

      {paymentMethod === "qris" ? (
        <div className="qris-panel">
          <div className="qris-code-card">
            <img alt="Static QRIS payment code" src="/qris/static-qris.png" />
          </div>
          <p>Check the customer's QRIS payment manually before confirming paid.</p>
          <p className="qris-panel__meta">This is a static QRIS image</p>
        </div>
      ) : (
        <p className="payment-panel__empty">Choose Cash or QRIS to prepare the payment confirmation.</p>
      )}
    </section>
  );
}

function hasRequiredModifiers(item: MenuItem, selectedModifiers: SelectedModifiers): boolean {
  return item.modifierGroups.every((group) => !group.required || Boolean(selectedModifiers[group.slug]));
}

function menuItemImageSrc(item: MenuItem): string {
  return item.slug.includes("latte") ? "/menu/latte.png" : "/menu/americano.png";
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
