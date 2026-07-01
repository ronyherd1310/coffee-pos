import { useEffect, useRef } from "preact/hooks";

type CancelOrderDialogProps = {
  isSubmitting: boolean;
  onBack: () => void;
  onConfirm: () => void;
};

export function CancelOrderDialog({ isSubmitting, onBack, onConfirm }: CancelOrderDialogProps) {
  const backButtonRef = useRef<HTMLButtonElement>(null);

  useEffect(() => {
    backButtonRef.current?.focus();
  }, []);

  return (
    <div className="dialog-backdrop">
      <section aria-labelledby="cancel-order-title" aria-modal="true" className="dialog-panel" role="dialog">
        <h2 id="cancel-order-title">Cancel order</h2>
        <p>This cancels the currently shown paid order. It does not delete local order history.</p>
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
            {isSubmitting ? "Cancelling..." : "Cancel Order"}
          </button>
        </div>
      </section>
    </div>
  );
}
