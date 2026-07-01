import { formatRupiah } from "../../lib/format";
import { formatModifierSummary, menuItemImageSrc } from "./cashierItemView";
import { calculateLineTotal, type CartLine } from "./orderDraft";
import type { PaymentMethod } from "./types";

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

export function CurrentOrderPanel({
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
