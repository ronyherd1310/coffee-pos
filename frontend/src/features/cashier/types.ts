export type PaymentMethod = "cash" | "qris";

export type PaidOrderStatus = "paid" | "cancelled";

export type ModifierOption = {
  name: string;
  slug: string;
  priceDeltaRp: number;
};

export type ModifierGroup = {
  name: string;
  slug: string;
  required: boolean;
  selectionType: "single";
  options: ModifierOption[];
};

export type MenuItem = {
  name: string;
  slug: string;
  priceRp: number;
  modifierGroups: ModifierGroup[];
};

export type MenuCategory = {
  name: string;
  slug: string;
  items: MenuItem[];
};

export type CashierMenu = {
  categories: MenuCategory[];
};

export type OrderLineInput = {
  menuItemSlug: string;
  quantity: number;
  modifiers: OrderModifierInput[];
};

export type OrderModifierInput = {
  groupSlug: string;
  optionSlug: string;
};

export type CreatePaidOrderInput = {
  clientRequestId: string;
  paymentMethod: PaymentMethod;
  note?: string;
  lines: OrderLineInput[];
};

export type PaidOrderDetail = {
  orderId: string;
  queueNumber: number;
  businessDate: string;
  status: PaidOrderStatus;
  paymentMethod: PaymentMethod;
  paidAt: string;
  cancelledAt: string | null;
  note: string | null;
  totalRp: number;
  lines: PaidOrderLine[];
};

export type PaidOrderLine = {
  menuItemSlug: string;
  menuItemName: string;
  unitPriceRp: number;
  quantity: number;
  lineTotalRp: number;
  modifiers: PaidOrderModifier[];
};

export type PaidOrderModifier = {
  groupSlug: string;
  groupName: string;
  optionSlug: string;
  optionName: string;
  priceDeltaRp: number;
};
