import { formatQueueNumber, formatRupiah } from "../../lib/format";
import type { PaidOrderDetail as PaidOrderDetailType } from "./types";

type PaidOrderDetailProps = {
  cancelError: string | undefined;
  order: PaidOrderDetailType;
  showPrintableTicket: boolean;
  onCancelClick: () => void;
  onPrintClick: () => void;
  onStartNew: () => void;
};

export function PaidOrderDetail({
  cancelError,
  order,
  showPrintableTicket,
  onCancelClick,
  onPrintClick,
  onStartNew
}: PaidOrderDetailProps) {
  const isCancelled = order.status === "cancelled";

  return (
    <>
      <section className="cashier-panel paid-detail-panel" aria-labelledby="paid-order-title">
        <div className="paid-detail-panel__header">
          <div>
            <h2 id="paid-order-title">Paid order created</h2>
            <p>{formatQueueNumber(order.queueNumber)}</p>
          </div>
          <p className={isCancelled ? "status-pill status-pill--cancelled" : "status-pill"}>
            Status: {isCancelled ? "Cancelled" : "Paid"}
          </p>
        </div>

        {cancelError ? (
          <p className="cashier-alert" role="alert">
            {cancelError}
          </p>
        ) : null}

        <div className="paid-detail-grid">
          <p>Payment: {order.paymentMethod === "qris" ? "QRIS" : "Cash"}</p>
          <p>Paid at: {order.paidAt}</p>
          {order.cancelledAt ? <p>Cancelled at: {order.cancelledAt}</p> : null}
          <p>Total: {formatRupiah(order.totalRp)}</p>
          {order.note ? <p>Note: {order.note}</p> : null}
        </div>

        <ul className="cart-lines" aria-label="Paid order items">
          {order.lines.map((line) => (
            <li className="cart-line" key={`${line.menuItemSlug}-${line.lineTotalRp}`}>
              <div>
                <p className="cart-line__name">{line.menuItemName}</p>
                <p>{line.modifiers.map((modifier) => modifier.optionName).join(", ")}</p>
                <p>
                  {line.quantity} x {formatRupiah(line.unitPriceRp)} = {formatRupiah(line.lineTotalRp)}
                </p>
              </div>
            </li>
          ))}
        </ul>

        <div className="payment-actions">
          <button className="button button--primary" disabled={isCancelled} onClick={onPrintClick} type="button">
            Print Ticket
          </button>
          {!isCancelled ? (
            <button className="button button--secondary" onClick={onCancelClick} type="button">
              Cancel Order
            </button>
          ) : null}
          <button className="button button--secondary" onClick={onStartNew} type="button">
            Start New
          </button>
        </div>
      </section>

      {showPrintableTicket ? <PrintableTicket order={order} /> : null}
    </>
  );
}

function PrintableTicket({ order }: { order: PaidOrderDetailType }) {
  return (
    <section aria-label="Printable ticket" className="printable-ticket">
      <h2>Ticket {formatQueueNumber(order.queueNumber)}</h2>
      <p>Paid at {order.paidAt}</p>
      <ul>
        {order.lines.map((line) => (
          <li key={`${line.menuItemSlug}-${line.quantity}`}>
            <span>
              {line.quantity} x {line.menuItemName}
            </span>
            <span>{line.modifiers.map((modifier) => modifier.optionName).join(", ")}</span>
            <span>{formatRupiah(line.lineTotalRp)}</span>
          </li>
        ))}
      </ul>
      {order.note ? <p>Note: {order.note}</p> : null}
      <p>Total {formatRupiah(order.totalRp)}</p>
      <p>Payment: {order.paymentMethod === "qris" ? "QRIS" : "Cash"}</p>
    </section>
  );
}
