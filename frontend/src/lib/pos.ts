import type {
  CashierMenu,
  CreatePaidOrderInput,
  MenuCategory,
  MenuItem,
  ModifierGroup,
  ModifierOption,
  PaidOrderDetail,
  PaidOrderLine,
  PaidOrderModifier,
  PaidOrderStatus,
  PaymentMethod
} from "../features/cashier/types";

export type CashierMenuResult =
  | { status: "success"; menu: CashierMenu }
  | { status: "unauthorized" }
  | { status: "unavailable" }
  | { status: "unexpected" };

export type CreatePaidOrderResult =
  | { status: "success"; order: PaidOrderDetail }
  | { status: "invalid-client-request-id" }
  | { status: "idempotency-conflict" }
  | { status: "invalid-order" }
  | { status: "unauthorized" }
  | { status: "unavailable" }
  | { status: "unexpected" };

export type CancelPaidOrderResult =
  | { status: "success"; order: PaidOrderDetail }
  | { status: "not-cancellable" }
  | { status: "not-found" }
  | { status: "unauthorized" }
  | { status: "unavailable" }
  | { status: "unexpected" };

type ErrorResponse = {
  error?: unknown;
};

export async function getCashierMenu(): Promise<CashierMenuResult> {
  try {
    const response = await fetch("/api/pos/menu", {
      credentials: "same-origin",
      headers: { Accept: "application/json" },
      method: "GET"
    });

    if (response.status === 401) {
      return { status: "unauthorized" };
    }

    if (!response.ok) {
      return { status: "unavailable" };
    }

    const menu = parseCashierMenu(await response.json());
    return menu ? { status: "success", menu } : { status: "unexpected" };
  } catch {
    return { status: "unavailable" };
  }
}

export async function createPaidOrder(input: CreatePaidOrderInput): Promise<CreatePaidOrderResult> {
  try {
    const response = await fetch("/api/pos/orders", {
      body: JSON.stringify(input),
      credentials: "same-origin",
      headers: {
        Accept: "application/json",
        "Content-Type": "application/json"
      },
      method: "POST"
    });

    if (response.ok) {
      const order = parsePaidOrderDetail(await response.json());
      return order ? { status: "success", order } : { status: "unexpected" };
    }

    const error = await readErrorCode(response);

    if (response.status === 400 && error === "invalid_client_request_id") {
      return { status: "invalid-client-request-id" };
    }

    if (response.status === 401) {
      return { status: "unauthorized" };
    }

    if (response.status === 409 && error === "idempotency_conflict") {
      return { status: "idempotency-conflict" };
    }

    if (response.status === 422 && error === "invalid_order") {
      return { status: "invalid-order" };
    }

    return { status: "unexpected" };
  } catch {
    return { status: "unavailable" };
  }
}

export async function cancelPaidOrder(orderId: string): Promise<CancelPaidOrderResult> {
  try {
    const response = await fetch(`/api/pos/orders/${encodeURIComponent(orderId)}/cancel`, {
      credentials: "same-origin",
      headers: { Accept: "application/json" },
      method: "POST"
    });

    if (response.ok) {
      const order = parsePaidOrderDetail(await response.json());
      return order ? { status: "success", order } : { status: "unexpected" };
    }

    const error = await readErrorCode(response);

    if (response.status === 401) {
      return { status: "unauthorized" };
    }

    if (response.status === 404 && error === "not_found") {
      return { status: "not-found" };
    }

    if (response.status === 409 && error === "order_not_cancellable") {
      return { status: "not-cancellable" };
    }

    return { status: "unexpected" };
  } catch {
    return { status: "unavailable" };
  }
}

async function readErrorCode(response: Response): Promise<string | undefined> {
  try {
    const data = (await response.json()) as ErrorResponse;
    return typeof data.error === "string" ? data.error : undefined;
  } catch {
    return undefined;
  }
}

function parseCashierMenu(data: unknown): CashierMenu | undefined {
  if (!isRecord(data) || !Array.isArray(data.categories)) {
    return undefined;
  }

  const categories = data.categories.map(parseMenuCategory);

  if (categories.some((category) => category === undefined)) {
    return undefined;
  }

  return { categories: categories as MenuCategory[] };
}

function parseMenuCategory(data: unknown): MenuCategory | undefined {
  if (!isRecord(data) || !isString(data.name) || !isString(data.slug) || !Array.isArray(data.items)) {
    return undefined;
  }

  const items = data.items.map(parseMenuItem);

  if (items.some((item) => item === undefined)) {
    return undefined;
  }

  return {
    items: items as MenuItem[],
    name: data.name,
    slug: data.slug
  };
}

function parseMenuItem(data: unknown): MenuItem | undefined {
  if (
    !isRecord(data) ||
    !isString(data.name) ||
    !isString(data.slug) ||
    !isInteger(data.priceRp) ||
    !Array.isArray(data.modifierGroups)
  ) {
    return undefined;
  }

  const modifierGroups = data.modifierGroups.map(parseModifierGroup);

  if (modifierGroups.some((group) => group === undefined)) {
    return undefined;
  }

  const displayMetadata = parseMenuItemDisplayMetadata(data);
  if (!displayMetadata) {
    return undefined;
  }

  return {
    ...displayMetadata,
    modifierGroups: modifierGroups as ModifierGroup[],
    name: data.name,
    priceRp: data.priceRp,
    slug: data.slug
  };
}

function parseMenuItemDisplayMetadata(data: Record<string, unknown>): Partial<MenuItem> | undefined {
  const metadata: Partial<MenuItem> = {};

  if (data.imagePath !== undefined) {
    if (!isString(data.imagePath)) {
      return undefined;
    }
    metadata.imagePath = data.imagePath;
  }

  if (data.popularityRank !== undefined) {
    if (!isInteger(data.popularityRank)) {
      return undefined;
    }
    metadata.popularityRank = data.popularityRank;
  }

  for (const key of ["bestSeller", "promo", "iced", "lowSugar", "newArrival"] as const) {
    if (data[key] === undefined) {
      continue;
    }
    if (typeof data[key] !== "boolean") {
      return undefined;
    }
    metadata[key] = data[key];
  }

  return metadata;
}

function parseModifierGroup(data: unknown): ModifierGroup | undefined {
  if (
    !isRecord(data) ||
    !isString(data.name) ||
    !isString(data.slug) ||
    typeof data.required !== "boolean" ||
    data.selectionType !== "single" ||
    !Array.isArray(data.options)
  ) {
    return undefined;
  }

  const options = data.options.map(parseModifierOption);

  if (options.some((option) => option === undefined)) {
    return undefined;
  }

  return {
    name: data.name,
    options: options as ModifierOption[],
    required: data.required,
    selectionType: "single",
    slug: data.slug
  };
}

function parseModifierOption(data: unknown): ModifierOption | undefined {
  if (!isRecord(data) || !isString(data.name) || !isString(data.slug) || !isInteger(data.priceDeltaRp)) {
    return undefined;
  }

  return {
    name: data.name,
    priceDeltaRp: data.priceDeltaRp,
    slug: data.slug
  };
}

function parsePaidOrderDetail(data: unknown): PaidOrderDetail | undefined {
  if (
    !isRecord(data) ||
    !isString(data.orderId) ||
    !isInteger(data.queueNumber) ||
    !isString(data.businessDate) ||
    !isPaidOrderStatus(data.status) ||
    !isPaymentMethod(data.paymentMethod) ||
    !isString(data.paidAt) ||
    !(data.cancelledAt === null || isString(data.cancelledAt)) ||
    !(data.note === null || isString(data.note)) ||
    !isInteger(data.totalRp) ||
    !Array.isArray(data.lines)
  ) {
    return undefined;
  }

  const lines = data.lines.map(parsePaidOrderLine);

  if (lines.some((line) => line === undefined)) {
    return undefined;
  }

  return {
    businessDate: data.businessDate,
    cancelledAt: data.cancelledAt,
    lines: lines as PaidOrderLine[],
    note: data.note,
    orderId: data.orderId,
    paidAt: data.paidAt,
    paymentMethod: data.paymentMethod,
    queueNumber: data.queueNumber,
    status: data.status,
    totalRp: data.totalRp
  };
}

function parsePaidOrderLine(data: unknown): PaidOrderLine | undefined {
  if (
    !isRecord(data) ||
    !isString(data.menuItemSlug) ||
    !isString(data.menuItemName) ||
    !isInteger(data.unitPriceRp) ||
    !isInteger(data.quantity) ||
    !isInteger(data.lineTotalRp) ||
    !Array.isArray(data.modifiers)
  ) {
    return undefined;
  }

  const modifiers = data.modifiers.map(parsePaidOrderModifier);

  if (modifiers.some((modifier) => modifier === undefined)) {
    return undefined;
  }

  return {
    lineTotalRp: data.lineTotalRp,
    menuItemName: data.menuItemName,
    menuItemSlug: data.menuItemSlug,
    modifiers: modifiers as PaidOrderModifier[],
    quantity: data.quantity,
    unitPriceRp: data.unitPriceRp
  };
}

function parsePaidOrderModifier(data: unknown): PaidOrderModifier | undefined {
  if (
    !isRecord(data) ||
    !isString(data.groupSlug) ||
    !isString(data.groupName) ||
    !isString(data.optionSlug) ||
    !isString(data.optionName) ||
    !isInteger(data.priceDeltaRp)
  ) {
    return undefined;
  }

  return {
    groupName: data.groupName,
    groupSlug: data.groupSlug,
    optionName: data.optionName,
    optionSlug: data.optionSlug,
    priceDeltaRp: data.priceDeltaRp
  };
}

function isPaidOrderStatus(value: unknown): value is PaidOrderStatus {
  return value === "paid" || value === "cancelled";
}

function isPaymentMethod(value: unknown): value is PaymentMethod {
  return value === "cash" || value === "qris";
}

function isInteger(value: unknown): value is number {
  return Number.isInteger(value);
}

function isString(value: unknown): value is string {
  return typeof value === "string";
}

function isRecord(value: unknown): value is Record<string, unknown> {
  return typeof value === "object" && value !== null && !Array.isArray(value);
}
