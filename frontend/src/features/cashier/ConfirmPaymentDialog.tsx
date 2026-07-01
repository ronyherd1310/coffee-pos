import { useEffect, useRef } from "preact/hooks";
import { formatRupiah } from "../../lib/format";
import type { PaymentMethod } from "./types";

type ConfirmPaymentDialogProps = {
  error: string | undefined;
  isSubmitting: boolean;
  paymentMethod: PaymentMethod | undefined;
  totalRp: number;
  onBack: () => void;
  onConfirm: () => void;
};

export function ConfirmPaymentDialog({
  error,
  isSubmitting,
  paymentMethod,
  totalRp,
  onBack,
  onConfirm
}: ConfirmPaymentDialogProps) {
  const backButtonRef = useRef<HTMLButtonElement>(null);

  useEffect(() => {
    backButtonRef.current?.focus();
  }, []);

  return (
    <div className="dialog-backdrop">
      <section
        aria-labelledby="confirm-payment-title"
        aria-modal="true"
        className="dialog-panel"
        role="dialog"
      >
        <h2 id="confirm-payment-title">Confirm payment</h2>
        <p>This will persist the order as paid. The order cannot be edited after confirmation.</p>
        <dl className="dialog-summary">
          <div>
            <dt>Total</dt>
            <dd>Total: {formatRupiah(totalRp)}</dd>
          </div>
          <div>
            <dt>Payment</dt>
            <dd>Payment: {paymentMethod === "qris" ? "QRIS" : "Cash"}</dd>
          </div>
        </dl>
        {error ? (
          <p className="cashier-alert" role="alert">
            {error}
          </p>
        ) : null}
        <div className="dialog-actions">
          <button
            className="button button--secondary"
            disabled={isSubmitting}
            onClick={onBack}
            ref={backButtonRef}
            type="button"
          >
            Back
          </button>
          <button
            className="button button--primary"
            disabled={isSubmitting}
            onClick={onConfirm}
            type="button"
          >
            {isSubmitting ? "Confirming..." : "Confirm Paid"}
          </button>
        </div>
      </section>
    </div>
  );
}
