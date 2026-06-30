export type SessionResult =
  | { status: "authenticated" }
  | { status: "unauthenticated" }
  | { status: "unavailable" };

export type LoginResult =
  | { status: "success" }
  | { status: "invalid-pin" }
  | { status: "rate-limited" }
  | { status: "unavailable" }
  | { status: "unknown-error" };

export type LogoutResult = { status: "success" } | { status: "unavailable" };

type ErrorResponse = {
  error?: unknown;
};

type SessionResponse = {
  authenticated?: unknown;
};

export async function getSession(): Promise<SessionResult> {
  try {
    const response = await fetch("/api/auth/session", {
      credentials: "same-origin",
      headers: {
        Accept: "application/json"
      },
      method: "GET"
    });

    if (!response.ok) {
      return { status: "unavailable" };
    }

    const data = (await response.json()) as SessionResponse;

    if (data.authenticated === true) {
      return { status: "authenticated" };
    }

    if (data.authenticated === false) {
      return { status: "unauthenticated" };
    }

    return { status: "unavailable" };
  } catch {
    return { status: "unavailable" };
  }
}

export async function loginWithPin(pin: string): Promise<LoginResult> {
  try {
    const response = await fetch("/api/auth/login", {
      body: JSON.stringify({ pin }),
      credentials: "same-origin",
      headers: {
        Accept: "application/json",
        "Content-Type": "application/json"
      },
      method: "POST"
    });

    if (response.ok) {
      return { status: "success" };
    }

    const error = await readErrorCode(response);

    if (response.status === 401 && error === "invalid_pin") {
      return { status: "invalid-pin" };
    }

    if (response.status === 429 && error === "too_many_attempts") {
      return { status: "rate-limited" };
    }

    return { status: "unknown-error" };
  } catch {
    return { status: "unavailable" };
  }
}

export async function logout(): Promise<LogoutResult> {
  try {
    const response = await fetch("/api/auth/logout", {
      credentials: "same-origin",
      headers: {
        Accept: "application/json"
      },
      method: "POST"
    });

    return response.ok ? { status: "success" } : { status: "unavailable" };
  } catch {
    return { status: "unavailable" };
  }
}

async function readErrorCode(response: Response): Promise<string | undefined> {
  try {
    const data = (await response.json()) as ErrorResponse;
    return typeof data.error === "string" ? data.error : undefined;
  } catch {
    return undefined;
  }
}
