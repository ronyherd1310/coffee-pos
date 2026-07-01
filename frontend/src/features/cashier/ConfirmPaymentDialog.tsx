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
  const paymentLabel = paymentMethod === "qris" ? "QRIS" : "Cash";
  const title = `Payment: ${paymentLabel}`;

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
        <div className="dialog-panel__header">
          <h2 id="confirm-payment-title">{title}</h2>
          <button
            aria-label="Close payment"
            className="dialog-close-button"
            disabled={isSubmitting}
            onClick={onBack}
            type="button"
          >
            <span aria-hidden="true">×</span>
          </button>
        </div>
        <div className="dialog-total">
          <span>Total Amount</span>
          <strong>{formatRupiah(totalRp)}</strong>
        </div>
        {paymentMethod === "qris" ? (
          <div className="qris-panel qris-panel--dialog">
            <div className="qris-code-card">
              <img alt="Static QRIS payment code" src="/qris/static-qris.png" />
            </div>
            <p>Scan the QR code using your e-wallet or mobile banking.</p>
          </div>
        ) : null}
        {error ? (
          <p className="cashier-alert" role="alert">
            {error}
          </p>
        ) : null}
        <div className="dialog-actions">
          <button
            className="button button--primary"
            disabled={isSubmitting}
            onClick={onConfirm}
            type="button"
          >
            {isSubmitting ? "Confirming..." : "Confirm Paid"}
          </button>
          <button
            className="button button--secondary"
            disabled={isSubmitting}
            onClick={onBack}
            ref={backButtonRef}
            type="button"
          >
            Cancel
          </button>
        </div>
      </section>
    </div>
  );
}
