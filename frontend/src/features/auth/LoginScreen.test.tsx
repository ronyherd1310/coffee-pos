import { fireEvent, render, screen, waitFor } from "@testing-library/preact";
import { afterEach, describe, expect, it, vi } from "vitest";
import { LoginScreen } from "./LoginScreen";

describe("LoginScreen", () => {
  afterEach(() => {
    vi.unstubAllGlobals();
  });

  it("keeps Sign In disabled until 6 digits are entered", () => {
    vi.stubGlobal("fetch", vi.fn());

    render(<LoginScreen onAuthenticated={vi.fn()} />);

    expect(screen.getByRole("button", { name: "Sign In" })).toBeDisabled();

    fireEvent.input(screen.getByLabelText("Cashier PIN"), {
      target: { value: "12345" }
    });

    expect(screen.getByRole("button", { name: "Sign In" })).toBeDisabled();

    fireEvent.input(screen.getByLabelText("Cashier PIN"), {
      target: { value: "123456" }
    });

    expect(screen.getByRole("button", { name: "Sign In" })).toBeEnabled();
  });

  it("submits with Enter when exactly 6 digits are present", async () => {
    const fetchMock = vi.fn().mockResolvedValue(new Response(null, { status: 204 }));
    const onAuthenticated = vi.fn();
    vi.stubGlobal("fetch", fetchMock);

    render(<LoginScreen onAuthenticated={onAuthenticated} />);

    fireEvent.input(screen.getByLabelText("Cashier PIN"), {
      target: { value: "123456" }
    });
    fireEvent.keyDown(screen.getByLabelText("Cashier PIN"), { key: "Enter" });

    await waitFor(() => expect(onAuthenticated).toHaveBeenCalledTimes(1));
    expect(fetchMock).toHaveBeenCalledWith("/api/auth/login", expect.objectContaining({ method: "POST" }));
  });

  it("updates auth state after successful login", async () => {
    const onAuthenticated = vi.fn();
    vi.stubGlobal("fetch", vi.fn().mockResolvedValue(new Response(null, { status: 200 })));

    render(<LoginScreen onAuthenticated={onAuthenticated} />);

    fireEvent.input(screen.getByLabelText("Cashier PIN"), {
      target: { value: "123456" }
    });
    fireEvent.click(screen.getByRole("button", { name: "Sign In" }));

    await waitFor(() => expect(onAuthenticated).toHaveBeenCalledTimes(1));
  });

  it("shows invalid PIN, clears the PIN, and returns focus to PIN entry", async () => {
    vi.stubGlobal(
      "fetch",
      vi.fn().mockResolvedValue(
        new Response(JSON.stringify({ error: "invalid_pin" }), {
          headers: { "Content-Type": "application/json" },
          status: 401
        })
      )
    );

    render(<LoginScreen onAuthenticated={vi.fn()} />);

    fireEvent.input(screen.getByLabelText("Cashier PIN"), {
      target: { value: "123456" }
    });
    fireEvent.click(screen.getByRole("button", { name: "Sign In" }));

    expect(await screen.findByRole("alert")).toHaveTextContent("Invalid PIN. Try again.");
    expect(screen.getByLabelText("Cashier PIN")).toHaveValue("");
    expect(screen.getByLabelText("Cashier PIN")).toHaveFocus();
  });

  it("shows the rate-limit message for too many attempts", async () => {
    vi.stubGlobal(
      "fetch",
      vi.fn().mockResolvedValue(
        new Response(JSON.stringify({ error: "too_many_attempts" }), {
          headers: { "Content-Type": "application/json" },
          status: 429
        })
      )
    );

    render(<LoginScreen onAuthenticated={vi.fn()} />);

    fireEvent.input(screen.getByLabelText("Cashier PIN"), {
      target: { value: "123456" }
    });
    fireEvent.click(screen.getByRole("button", { name: "Sign In" }));

    expect(await screen.findByRole("alert")).toHaveTextContent(
      "Too many attempts. Try again in a few minutes."
    );
  });

  it("shows a backend-unavailable message when login cannot reach the API", async () => {
    vi.stubGlobal("fetch", vi.fn().mockRejectedValue(new TypeError("network")));

    render(<LoginScreen onAuthenticated={vi.fn()} />);

    fireEvent.input(screen.getByLabelText("Cashier PIN"), {
      target: { value: "123456" }
    });
    fireEvent.click(screen.getByRole("button", { name: "Sign In" }));

    expect(await screen.findByRole("alert")).toHaveTextContent(
      "Backend unavailable. Check the connection and try again."
    );
    expect(screen.getByLabelText("Cashier PIN")).toHaveValue("");
  });

  it("keeps a stable loading label while a login request is pending", async () => {
    let resolveLogin: (response: Response) => void = () => undefined;
    vi.stubGlobal(
      "fetch",
      vi.fn().mockReturnValue(new Promise((resolve) => {
        resolveLogin = resolve;
      }))
    );

    render(<LoginScreen onAuthenticated={vi.fn()} />);

    fireEvent.input(screen.getByLabelText("Cashier PIN"), {
      target: { value: "123456" }
    });
    fireEvent.click(screen.getByRole("button", { name: "Sign In" }));

    expect(screen.getByRole("button", { name: "Signing in..." })).toBeDisabled();

    resolveLogin(new Response(null, { status: 204 }));
    await waitFor(() => expect(screen.queryByRole("button", { name: "Signing in..." })).not.toBeInTheDocument());
  });
});
