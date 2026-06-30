import { fireEvent, render, screen } from "@testing-library/preact";
import { afterEach, describe, expect, it, vi } from "vitest";
import { App } from "./App";

describe("App auth bootstrap", () => {
  afterEach(() => {
    vi.unstubAllGlobals();
  });

  it("shows a loading state while checking the current session", () => {
    vi.stubGlobal("fetch", vi.fn().mockReturnValue(new Promise(() => undefined)));

    render(<App />);

    expect(screen.getByRole("status", { name: "Checking session" })).toHaveTextContent(
      "Checking session..."
    );
    expect(screen.queryByText("Protected POS shell")).not.toBeInTheDocument();
  });

  it("renders the cashier PIN screen when the session is unauthenticated", async () => {
    vi.stubGlobal(
      "fetch",
      vi.fn().mockResolvedValue(
        new Response(JSON.stringify({ authenticated: false }), {
          headers: { "Content-Type": "application/json" },
          status: 200
        })
      )
    );

    render(<App />);

    expect(await screen.findByRole("heading", { level: 1, name: "Coffee POS" })).toBeVisible();
    expect(screen.getByText("Sign in to continue")).toBeVisible();
    expect(screen.getByText("Cashier PIN")).toBeVisible();
    expect(screen.queryByText("Protected POS shell")).not.toBeInTheDocument();
  });

  it("renders protected content when the session is authenticated", async () => {
    vi.stubGlobal(
      "fetch",
      vi.fn().mockResolvedValue(
        new Response(JSON.stringify({ authenticated: true }), {
          headers: { "Content-Type": "application/json" },
          status: 200
        })
      )
    );

    render(<App />);

    expect(await screen.findByText("Protected POS shell")).toBeVisible();
    expect(screen.queryByText("Cashier PIN")).not.toBeInTheDocument();
  });

  it("returns to the cashier PIN screen after logout succeeds", async () => {
    const fetchMock = vi
      .fn()
      .mockResolvedValueOnce(
        new Response(JSON.stringify({ authenticated: true }), {
          headers: { "Content-Type": "application/json" },
          status: 200
        })
      )
      .mockResolvedValueOnce(new Response(null, { status: 204 }));
    vi.stubGlobal("fetch", fetchMock);

    render(<App />);

    expect(await screen.findByText("Protected POS shell")).toBeVisible();

    fireEvent.click(screen.getByRole("button", { name: "Logout" }));

    expect(await screen.findByText("Cashier PIN")).toBeVisible();
    expect(screen.queryByText("Protected POS shell")).not.toBeInTheDocument();
  });

  it("renders a recoverable error state when the session check fails", async () => {
    const fetchMock = vi
      .fn()
      .mockRejectedValueOnce(new TypeError("network"))
      .mockResolvedValueOnce(
        new Response(JSON.stringify({ authenticated: false }), {
          headers: { "Content-Type": "application/json" },
          status: 200
        })
      );
    vi.stubGlobal("fetch", fetchMock);

    render(<App />);

    expect(await screen.findByRole("alert")).toHaveTextContent(
      "Cannot check the current session."
    );
    expect(screen.queryByText("Protected POS shell")).not.toBeInTheDocument();

    fireEvent.click(screen.getByRole("button", { name: "Retry" }));

    expect(await screen.findByText("Cashier PIN")).toBeVisible();
    expect(fetchMock).toHaveBeenCalledTimes(2);
  });
});
