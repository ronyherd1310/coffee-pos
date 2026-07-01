import { useState } from "preact/hooks";
import { CashierOrderScreen } from "../cashier/CashierOrderScreen";
import { logout } from "../../lib/auth";

type ProtectedPosShellProps = {
  onSignedOut: () => void;
};

export function ProtectedPosShell({ onSignedOut }: ProtectedPosShellProps) {
  const [isLoggingOut, setIsLoggingOut] = useState(false);
  const [error, setError] = useState<string | undefined>();

  async function handleLogout() {
    setIsLoggingOut(true);
    setError(undefined);

    const result = await logout();

    setIsLoggingOut(false);

    if (result.status === "success") {
      onSignedOut();
      return;
    }

    setError("Cannot log out right now. Check the connection and try again.");
  }

  return (
    <main className="pos-shell">
      <header className="pos-shell__header">
        <div className="pos-shell__brand">
          <span className="pos-shell__mark" aria-hidden="true">
            POS
          </span>
          <div>
            <p className="pos-shell__status">Access active</p>
            <h1>Coffee POS</h1>
          </div>
        </div>

        <p className="sr-only">Search remains in the order catalog</p>

        <div className="pos-shell__actions">
          <nav className="pos-shell__nav" aria-label="POS sections">
            <a href="#new-order">New Order</a>
            <a href="#daily-summary">Today's Orders</a>
          </nav>
          <button
            aria-label={isLoggingOut ? "Logging out cashier" : "Logout cashier"}
            className="button button--secondary user-avatar-button"
            disabled={isLoggingOut}
            onClick={() => void handleLogout()}
            type="button"
          >
            <span className="user-avatar-button__label">{isLoggingOut ? "Logging out..." : "Cashier"}</span>
            <span className="user-avatar-button__status" aria-hidden="true" />
          </button>
        </div>
      </header>

      {error ? (
        <p className="pos-shell__error" role="alert">
          {error}
        </p>
      ) : null}

      <CashierOrderScreen onSessionExpired={onSignedOut} />
    </main>
  );
}
