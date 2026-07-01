import { fireEvent, render, screen, waitFor } from "@testing-library/preact";
import { afterEach, describe, expect, it, vi } from "vitest";
import { ProtectedPosShell } from "./ProtectedPosShell";

describe("ProtectedPosShell", () => {
  afterEach(() => {
    vi.unstubAllGlobals();
  });

  it("renders guarded POS shell landmarks and cashier navigation", () => {
    vi.stubGlobal("fetch", vi.fn().mockReturnValue(new Promise(() => undefined)));

    render(<ProtectedPosShell onSignedOut={vi.fn()} />);

    expect(screen.getByRole("heading", { level: 1, name: "Coffee POS" })).toBeVisible();
    expect(screen.getByText("POS")).toBeVisible();
    expect(screen.getByText("Search remains in the order catalog")).toHaveClass("sr-only");
    expect(screen.getByRole("link", { name: "New Order" })).toBeVisible();
    expect(screen.getByRole("link", { name: "Today's Orders" })).toBeVisible();
    expect(screen.getByRole("button", { name: "Logout cashier" })).toBeVisible();
    expect(screen.queryByText("Protected POS shell")).not.toBeInTheDocument();
  });

  it("calls backend logout and reports signed-out state on success", async () => {
    const fetchMock = vi.fn((url: RequestInfo | URL) => {
      if (String(url) === "/api/auth/logout") {
        return Promise.resolve(new Response(null, { status: 204 }));
      }

      return Promise.resolve(
        new Response(JSON.stringify({ categories: [] }), {
          headers: { "Content-Type": "application/json" },
          status: 200
        })
      );
    });
    const onSignedOut = vi.fn();
    vi.stubGlobal("fetch", fetchMock);

    render(<ProtectedPosShell onSignedOut={onSignedOut} />);

    fireEvent.click(screen.getByRole("button", { name: "Logout cashier" }));

    await waitFor(() => expect(onSignedOut).toHaveBeenCalledTimes(1));
    expect(fetchMock).toHaveBeenCalledWith("/api/auth/logout", {
      credentials: "same-origin",
      headers: {
        Accept: "application/json"
      },
      method: "POST"
    });
  });

  it("shows a recoverable error and stays signed in when logout fails", async () => {
    const onSignedOut = vi.fn();
    vi.stubGlobal(
      "fetch",
      vi.fn((url: RequestInfo | URL) => {
        if (String(url) === "/api/auth/logout") {
          return Promise.reject(new TypeError("network"));
        }

        return Promise.resolve(
          new Response(JSON.stringify({ categories: [] }), {
            headers: { "Content-Type": "application/json" },
            status: 200
          })
        );
      })
    );

    render(<ProtectedPosShell onSignedOut={onSignedOut} />);

    fireEvent.click(screen.getByRole("button", { name: "Logout cashier" }));

    expect(await screen.findByRole("alert")).toHaveTextContent(
      "Cannot log out right now. Check the connection and try again."
    );
    expect(onSignedOut).not.toHaveBeenCalled();
    expect(screen.getByRole("button", { name: "Logout cashier" })).toBeEnabled();
  });
});
