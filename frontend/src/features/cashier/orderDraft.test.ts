import { describe, expect, it } from "vitest";
import type { MenuItem } from "./types";
import {
  buildCreatePaidOrderPayload,
  calculateDraftBreakdown,
  calculateDraftTotal,
  calculateLineTotal,
  createCartLine,
  validateDraft
} from "./orderDraft";

const americano: MenuItem = {
  modifierGroups: [
    {
      name: "Temperature",
      options: [
        { name: "Hot", priceDeltaRp: 0, slug: "hot" },
        { name: "Iced", priceDeltaRp: 2000, slug: "iced" }
      ],
      required: true,
      selectionType: "single",
      slug: "temperature"
    },
    {
      name: "Sugar",
      options: [
        { name: "Normal", priceDeltaRp: 0, slug: "normal" },
        { name: "Less", priceDeltaRp: 0, slug: "less" }
      ],
      required: true,
      selectionType: "single",
      slug: "sugar"
    }
  ],
  name: "Americano",
  priceRp: 18000,
  slug: "americano"
};

describe("order draft helpers", () => {
  it("requires at least one cart line and a payment method", () => {
    expect(validateDraft({ lines: [], note: "", paymentMethod: undefined })).toEqual({
      isValid: false,
      errors: ["Add at least one item.", "Choose Cash or QRIS payment."]
    });
  });

  it("requires selected options for every required modifier group", () => {
    const line = createCartLine({
      id: "line-1",
      item: americano,
      quantity: 1,
      selectedModifiers: { temperature: "hot" }
    });

    expect(validateDraft({ lines: [line], note: "", paymentMethod: "cash" })).toEqual({
      isValid: false,
      errors: ["Choose Sugar for Americano."]
    });
  });

  it("rejects quantities outside 1 through 99", () => {
    const line = createCartLine({
      id: "line-1",
      item: americano,
      quantity: 100,
      selectedModifiers: { sugar: "normal", temperature: "hot" }
    });

    expect(validateDraft({ lines: [line], note: "", paymentMethod: "cash" })).toEqual({
      isValid: false,
      errors: ["Americano quantity must be between 1 and 99."]
    });
  });

  it("calculates line and draft totals from backend menu prices and modifier deltas", () => {
    const line = createCartLine({
      id: "line-1",
      item: americano,
      quantity: 2,
      selectedModifiers: { sugar: "normal", temperature: "iced" }
    });

    expect(calculateLineTotal(line)).toBe(40000);
    expect(calculateDraftTotal([line])).toBe(40000);
  });

  it("returns an 11% tax breakdown for the order summary", () => {
    const line = createCartLine({
      id: "line-1",
      item: americano,
      quantity: 2,
      selectedModifiers: { sugar: "normal", temperature: "iced" }
    });

    expect(calculateDraftBreakdown([line])).toEqual({
      subtotalRp: 40000,
      taxRp: 4400,
      totalRp: 44400
    });
  });

  it("keeps the same drink with different modifiers as separate cart lines", () => {
    const hotLine = createCartLine({
      id: "line-hot",
      item: americano,
      quantity: 1,
      selectedModifiers: { sugar: "normal", temperature: "hot" }
    });
    const icedLine = createCartLine({
      id: "line-iced",
      item: americano,
      quantity: 1,
      selectedModifiers: { sugar: "less", temperature: "iced" }
    });

    expect([hotLine, icedLine]).toHaveLength(2);
    expect(hotLine.id).not.toBe(icedLine.id);
    expect(hotLine.selectedModifiers).not.toEqual(icedLine.selectedModifiers);
  });

  it("builds create-order payload without server-owned fields and omits empty notes", () => {
    const line = createCartLine({
      id: "line-1",
      item: americano,
      quantity: 2,
      selectedModifiers: { sugar: "less", temperature: "iced" }
    });

    const payload = buildCreatePaidOrderPayload({
        clientRequestId: "11111111-1111-4111-8111-111111111111",
        draft: { lines: [line], note: "   ", paymentMethod: "qris" }
    });

    expect(payload).toEqual({
      clientRequestId: "11111111-1111-4111-8111-111111111111",
      lines: [
        {
          menuItemSlug: "americano",
          modifiers: [
            { groupSlug: "temperature", optionSlug: "iced" },
            { groupSlug: "sugar", optionSlug: "less" }
          ],
          quantity: 2
        }
      ],
      paymentMethod: "qris"
    });
    expect(payload).not.toHaveProperty("subtotalRp");
    expect(payload).not.toHaveProperty("taxRp");
    expect(payload).not.toHaveProperty("totalRp");
  });
});
