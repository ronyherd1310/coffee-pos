import { describe, expect, it } from "vitest";
import { formatQueueNumber, formatRupiah } from "./format";

describe("cashier formatting helpers", () => {
  it("formats integer rupiah without cents", () => {
    expect(formatRupiah(43000)).toBe("Rp43.000");
    expect(formatRupiah(1000)).toBe("Rp1.000");
  });

  it("formats queue numbers with three digits", () => {
    expect(formatQueueNumber(1)).toBe("Queue No. 001");
    expect(formatQueueNumber(42)).toBe("Queue No. 042");
  });
});
