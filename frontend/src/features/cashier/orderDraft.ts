import type {
  CreatePaidOrderInput,
  MenuItem,
  OrderLineInput,
  OrderModifierInput,
  PaymentMethod
} from "./types";

export type SelectedModifiers = Record<string, string>;

export type CartLine = {
  id: string;
  item: MenuItem;
  quantity: number;
  selectedModifiers: SelectedModifiers;
};

export type DraftOrder = {
  lines: CartLine[];
  note: string;
  paymentMethod: PaymentMethod | undefined;
};

export type DraftValidation = { isValid: true; errors: [] } | { isValid: false; errors: string[] };

export function createCartLine(input: CartLine): CartLine {
  return {
    id: input.id,
    item: input.item,
    quantity: input.quantity,
    selectedModifiers: { ...input.selectedModifiers }
  };
}

export function validateDraft(draft: DraftOrder): DraftValidation {
  const errors: string[] = [];

  if (draft.lines.length === 0) {
    errors.push("Add at least one item.");
  }

  for (const line of draft.lines) {
    if (line.quantity < 1 || line.quantity > 99) {
      errors.push(`${line.item.name} quantity must be between 1 and 99.`);
    }

    for (const group of line.item.modifierGroups) {
      if (group.required && !line.selectedModifiers[group.slug]) {
        errors.push(`Choose ${group.name} for ${line.item.name}.`);
      }
    }
  }

  if (!draft.paymentMethod) {
    errors.push("Choose Cash or QRIS payment.");
  }

  return errors.length > 0 ? { errors, isValid: false } : { errors: [], isValid: true };
}

export function calculateLineTotal(line: CartLine): number {
  const modifiersTotal = line.item.modifierGroups.reduce((total, group) => {
    const selectedSlug = line.selectedModifiers[group.slug];
    const selectedOption = group.options.find((option) => option.slug === selectedSlug);
    return total + (selectedOption?.priceDeltaRp ?? 0);
  }, 0);

  return (line.item.priceRp + modifiersTotal) * line.quantity;
}

export function calculateDraftTotal(lines: CartLine[]): number {
  return lines.reduce((total, line) => total + calculateLineTotal(line), 0);
}

export function buildCreatePaidOrderPayload(input: {
  clientRequestId: string;
  draft: DraftOrder;
}): CreatePaidOrderInput {
  const trimmedNote = input.draft.note.trim();
  const payload: CreatePaidOrderInput = {
    clientRequestId: input.clientRequestId,
    lines: input.draft.lines.map(toOrderLineInput),
    paymentMethod: input.draft.paymentMethod ?? "cash"
  };

  if (trimmedNote.length > 0) {
    payload.note = trimmedNote;
  }

  return payload;
}

export function clampQuantity(quantity: number): number {
  return Math.min(99, Math.max(1, quantity));
}

function toOrderLineInput(line: CartLine): OrderLineInput {
  return {
    menuItemSlug: line.item.slug,
    modifiers: toOrderModifiers(line),
    quantity: line.quantity
  };
}

function toOrderModifiers(line: CartLine): OrderModifierInput[] {
  return line.item.modifierGroups.flatMap((group) => {
    const optionSlug = line.selectedModifiers[group.slug];

    if (!optionSlug) {
      return [];
    }

    return [{ groupSlug: group.slug, optionSlug }];
  });
}
