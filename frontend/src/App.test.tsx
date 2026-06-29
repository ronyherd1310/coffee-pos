import { render, screen } from "@testing-library/preact";
import { afterEach, describe, expect, it, vi } from "vitest";
import { App } from "./App";

describe("App", () => {
  afterEach(() => {
    vi.unstubAllGlobals();
  });

  it("renders the Coffee POS title", () => {
    vi.stubGlobal(
      "fetch",
      vi.fn().mockResolvedValue(
        new Response(JSON.stringify({ status: "ok", service: "coffee-pos-backend" }), {
          headers: { "Content-Type": "application/json" },
          status: 200
        })
      )
    );

    render(<App />);

    expect(screen.getByRole("heading", { level: 1, name: "Coffee POS" })).toBeVisible();
  });

  it("shows backend health after the health check succeeds", async () => {
    vi.stubGlobal(
      "fetch",
      vi.fn().mockResolvedValue(
        new Response(JSON.stringify({ status: "ok", service: "coffee-pos-backend" }), {
          headers: { "Content-Type": "application/json" },
          status: 200
        })
      )
    );

    render(<App />);

    expect(await screen.findByText("coffee-pos-backend")).toBeVisible();
    expect(screen.getByText("ok")).toBeVisible();
  });
});
