import { fireEvent, render, screen, waitFor } from "@testing-library/preact";
import { afterEach, describe, expect, it, vi } from "vitest";
import { ProtectedPosShell } from "./ProtectedPosShell";

describe("ProtectedPosShell", () => {
  afterEach(() => {
    vi.unstubAllGlobals();
  });

  it("renders guarded POS shell landmarks and placeholder navigation", () => {
    vi.stubGlobal("fetch", vi.fn());

    render(<ProtectedPosShell onSignedOut={vi.fn()} />);

    expect(screen.getByRole("heading", { level: 1, name: "Coffee POS" })).toBeVisible();
    expect(screen.getByText("Access active")).toBeVisible();
    expect(screen.getByRole("link", { name: "New Order" })).toBeVisible();
    expect(screen.getByRole("link", { name: "Daily Summary" })).toBeVisible();
    expect(screen.getByRole("button", { name: "Logout" })).toBeVisible();
  });

  it("calls backend logout and reports signed-out state on success", async () => {
    const fetchMock = vi.fn().mockResolvedValue(new Response(null, { status: 204 }));
    const onSignedOut = vi.fn();
    vi.stubGlobal("fetch", fetchMock);

    render(<ProtectedPosShell onSignedOut={onSignedOut} />);

    fireEvent.click(screen.getByRole("button", { name: "Logout" }));

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
    vi.stubGlobal("fetch", vi.fn().mockRejectedValue(new TypeError("network")));

    render(<ProtectedPosShell onSignedOut={onSignedOut} />);

    fireEvent.click(screen.getByRole("button", { name: "Logout" }));

    expect(await screen.findByRole("alert")).toHaveTextContent(
      "Cannot log out right now. Check the connection and try again."
    );
    expect(onSignedOut).not.toHaveBeenCalled();
    expect(screen.getByRole("button", { name: "Logout" })).toBeEnabled();
  });
});
