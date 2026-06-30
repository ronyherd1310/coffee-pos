import { afterEach, describe, expect, it, vi } from "vitest";
import { getSession, loginWithPin, logout } from "./auth";

describe("auth API client", () => {
  afterEach(() => {
    vi.unstubAllGlobals();
  });

  it("returns authenticated session state from the relative session endpoint", async () => {
    const fetchMock = vi.fn().mockResolvedValue(
      new Response(JSON.stringify({ authenticated: true }), {
        headers: { "Content-Type": "application/json" },
        status: 200
      })
    );
    vi.stubGlobal("fetch", fetchMock);

    await expect(getSession()).resolves.toEqual({ status: "authenticated" });

    expect(fetchMock).toHaveBeenCalledWith("/api/auth/session", {
      credentials: "same-origin",
      headers: {
        Accept: "application/json"
      },
      method: "GET"
    });
  });

  it("returns unauthenticated session state as an expected result", async () => {
    vi.stubGlobal(
      "fetch",
      vi.fn().mockResolvedValue(
        new Response(JSON.stringify({ authenticated: false }), {
          headers: { "Content-Type": "application/json" },
          status: 200
        })
      )
    );

    await expect(getSession()).resolves.toEqual({ status: "unauthenticated" });
  });

  it("maps session network failures to unavailable", async () => {
    vi.stubGlobal("fetch", vi.fn().mockRejectedValue(new TypeError("failed")));

    await expect(getSession()).resolves.toEqual({ status: "unavailable" });
  });

  it("posts a cashier PIN as JSON and accepts any successful login response", async () => {
    const fetchMock = vi.fn().mockResolvedValue(new Response(null, { status: 204 }));
    vi.stubGlobal("fetch", fetchMock);

    await expect(loginWithPin("123456")).resolves.toEqual({ status: "success" });

    expect(fetchMock).toHaveBeenCalledWith("/api/auth/login", {
      body: JSON.stringify({ pin: "123456" }),
      credentials: "same-origin",
      headers: {
        Accept: "application/json",
        "Content-Type": "application/json"
      },
      method: "POST"
    });
  });

  it("maps invalid PIN login responses to a frontend-safe result", async () => {
    vi.stubGlobal(
      "fetch",
      vi.fn().mockResolvedValue(
        new Response(JSON.stringify({ error: "invalid_pin" }), {
          headers: { "Content-Type": "application/json" },
          status: 401
        })
      )
    );

    await expect(loginWithPin("123456")).resolves.toEqual({ status: "invalid-pin" });
  });

  it("maps rate-limited login responses to a frontend-safe result", async () => {
    vi.stubGlobal(
      "fetch",
      vi.fn().mockResolvedValue(
        new Response(JSON.stringify({ error: "too_many_attempts" }), {
          headers: { "Content-Type": "application/json" },
          status: 429
        })
      )
    );

    await expect(loginWithPin("123456")).resolves.toEqual({ status: "rate-limited" });
  });

  it("maps unexpected login responses to unknown-error", async () => {
    vi.stubGlobal(
      "fetch",
      vi.fn().mockResolvedValue(
        new Response(JSON.stringify({ error: "not_json_contract" }), {
          headers: { "Content-Type": "application/json" },
          status: 500
        })
      )
    );

    await expect(loginWithPin("123456")).resolves.toEqual({ status: "unknown-error" });
  });

  it("maps login network failures to unavailable", async () => {
    vi.stubGlobal("fetch", vi.fn().mockRejectedValue(new TypeError("failed")));

    await expect(loginWithPin("123456")).resolves.toEqual({ status: "unavailable" });
  });

  it("posts logout with same-origin credentials and accepts successful responses", async () => {
    const fetchMock = vi.fn().mockResolvedValue(new Response(null, { status: 204 }));
    vi.stubGlobal("fetch", fetchMock);

    await expect(logout()).resolves.toEqual({ status: "success" });

    expect(fetchMock).toHaveBeenCalledWith("/api/auth/logout", {
      credentials: "same-origin",
      headers: {
        Accept: "application/json"
      },
      method: "POST"
    });
  });

  it("maps logout network failures to unavailable", async () => {
    vi.stubGlobal("fetch", vi.fn().mockRejectedValue(new TypeError("failed")));

    await expect(logout()).resolves.toEqual({ status: "unavailable" });
  });
});
