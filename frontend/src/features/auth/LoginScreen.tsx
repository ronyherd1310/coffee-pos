import { useEffect, useRef, useState } from "preact/hooks";
import { loginWithPin } from "../../lib/auth";
import { PinInput } from "./PinInput";

type LoginScreenProps = {
  onAuthenticated: () => void;
};

const PIN_LENGTH = 6;

export function LoginScreen({ onAuthenticated }: LoginScreenProps) {
  const [pin, setPin] = useState("");
  const [error, setError] = useState<string | undefined>();
  const [isSubmitting, setIsSubmitting] = useState(false);
  const pinInputRef = useRef<HTMLInputElement>(null);
  const canSubmit = pin.length === PIN_LENGTH && !isSubmitting;

  useEffect(() => {
    if (error) {
      pinInputRef.current?.focus();
    }
  }, [error]);

  async function submitLogin() {
    if (!canSubmit) {
      return;
    }

    setIsSubmitting(true);
    setError(undefined);

    const result = await loginWithPin(pin);

    setIsSubmitting(false);

    if (result.status === "success") {
      setPin("");
      onAuthenticated();
      return;
    }

    setPin("");
    setError(messageForLoginFailure(result.status));
    queueMicrotask(() => pinInputRef.current?.focus());
  }

  return (
    <main className="login-page">
      <section className="login-panel" aria-labelledby="login-title">
        <div className="brand-mark" aria-hidden="true">
          <svg viewBox="0 0 48 48" focusable="false">
            <path d="M15 17h18l-2 24H17L15 17Z" />
            <path d="M13 13h22a3 3 0 0 1 3 3v1H10v-1a3 3 0 0 1 3-3Z" />
            <path d="M18 13V9h12v4" />
            <path d="M24 25c4 2 4 7 0 10-4-3-4-8 0-10Z" />
          </svg>
        </div>
        <h1 id="login-title">Coffee POS</h1>
        <p className="login-subtitle">Sign in to continue</p>
        <hr className="login-divider" />

        <form
          aria-label="Sign in"
          className="login-form"
          onSubmit={(event) => {
            event.preventDefault();
            void submitLogin();
          }}
        >
          <PinInput
            disabled={isSubmitting}
            describedById={error ? "login-error" : undefined}
            id="cashier-pin"
            helperText="Enter your 6-digit PIN"
            inputRef={pinInputRef}
            invalid={Boolean(error)}
            label="Cashier PIN"
            onChange={setPin}
            onKeyDown={(event) => {
              if (event.key === "Enter" && canSubmit) {
                event.preventDefault();
                void submitLogin();
              }
            }}
            value={pin}
          />
          <button className="button button--primary login-submit" disabled={!canSubmit} type="submit">
            {isSubmitting ? "Signing in..." : "Sign In"}
          </button>
        </form>

        <div className="login-or" aria-hidden="true">
          <span />
          <strong>or</strong>
          <span />
        </div>

        {error ? (
          <div className="login-alert" id="login-error" role="alert">
            <span className="login-alert__icon" aria-hidden="true">
              !
            </span>
            <span>{error}</span>
          </div>
        ) : null}
      </section>
    </main>
  );
}

function messageForLoginFailure(status: "invalid-pin" | "rate-limited" | "unavailable" | "unknown-error") {
  if (status === "invalid-pin") {
    return "Invalid PIN. Try again.";
  }

  if (status === "rate-limited") {
    return "Too many attempts. Try again in a few minutes.";
  }

  if (status === "unavailable") {
    return "Backend unavailable. Check the connection and try again.";
  }

  return "Cannot sign in right now. Try again.";
}
