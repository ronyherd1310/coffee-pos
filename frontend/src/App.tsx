import { LoginScreen } from "./features/auth/LoginScreen";
import { useAuthSession } from "./features/auth/useAuthSession";
import { ProtectedPosShell } from "./features/pos/ProtectedPosShell";

export function App() {
  const auth = useAuthSession();

  if (auth.state.status === "loading") {
    return (
      <main className="session-page">
        <p className="status-message" role="status" aria-label="Checking session">
          Checking session...
        </p>
      </main>
    );
  }

  if (auth.state.status === "unavailable") {
    return (
      <main className="session-page">
        <section className="session-panel" aria-labelledby="session-error-title">
          <h1 id="session-error-title">Coffee POS</h1>
          <p className="status-message status-message--error" role="alert">
            Cannot check the current session.
          </p>
          <button className="button button--primary" type="button" onClick={() => auth.refresh()}>
            Retry
          </button>
        </section>
      </main>
    );
  }

  if (auth.state.status === "authenticated") {
    return <ProtectedPosShell onSignedOut={auth.markUnauthenticated} />;
  }

  return <LoginScreen onAuthenticated={auth.markAuthenticated} />;
}
