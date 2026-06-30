import { useState } from "preact/hooks";
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
        <div>
          <p className="pos-shell__status">Access active</p>
          <h1>Coffee POS</h1>
        </div>
        <button
          className="button button--secondary"
          disabled={isLoggingOut}
          onClick={() => void handleLogout()}
          type="button"
        >
          {isLoggingOut ? "Logging out..." : "Logout"}
        </button>
      </header>

      <nav className="pos-shell__nav" aria-label="POS sections">
        <a href="#new-order">New Order</a>
        <a href="#daily-summary">Daily Summary</a>
      </nav>

      {error ? (
        <p className="pos-shell__error" role="alert">
          {error}
        </p>
      ) : null}

      <section className="pos-shell__placeholder" aria-labelledby="pos-placeholder-title">
        <h2 id="pos-placeholder-title">Protected POS shell</h2>
        <p>Order entry and daily summary workflows will attach here in later MVP slices.</p>
      </section>
    </main>
  );
}
