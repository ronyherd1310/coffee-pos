import { afterEach, describe, expect, it, vi } from "vitest";
import { fetchHealth } from "./health";

describe("fetchHealth", () => {
  afterEach(() => {
    vi.unstubAllGlobals();
  });

  it("returns backend health JSON from the relative health endpoint", async () => {
    const fetchMock = vi.fn().mockResolvedValue(
      new Response(JSON.stringify({ status: "ok", service: "coffee-pos-backend" }), {
        headers: { "Content-Type": "application/json" },
        status: 200
      })
    );
    vi.stubGlobal("fetch", fetchMock);

    await expect(fetchHealth()).resolves.toEqual({
      status: "ok",
      service: "coffee-pos-backend"
    });

    expect(fetchMock).toHaveBeenCalledWith("/api/health", {
      headers: {
        Accept: "application/json"
      }
    });
  });
});
