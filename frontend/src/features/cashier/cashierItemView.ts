import type { CartLine, SelectedModifiers } from "./orderDraft";
import type { MenuItem } from "./types";

export function hasRequiredModifiers(item: MenuItem, selectedModifiers: SelectedModifiers): boolean {
  return item.modifierGroups.every((group) => !group.required || Boolean(selectedModifiers[group.slug]));
}

export function hasRequiredModifierGroups(item: MenuItem): boolean {
  return item.modifierGroups.some((group) => group.required);
}

export function menuItemImageSrc(item: MenuItem): string {
  if (item.imagePath) {
    return item.imagePath;
  }

  const fallbackBySlug: Record<string, string> = {
    americano: "/menu/americano.png",
    latte: "/menu/latte.png"
  };

  return fallbackBySlug[item.slug] ?? "/menu/americano.png";
}

export function menuItemBadges(item: MenuItem): string[] {
  if (item.bestSeller) {
    return ["Best Seller"];
  }
  if (item.newArrival) {
    return ["New Arrival"];
  }
  if (item.promo) {
    return ["Promo"];
  }
  return [];
}

export function optionControlClass(optionSlug: string): string {
  return `option-control option-control--${optionSlug}`;
}

export function formatModifierSummary(line: CartLine): string {
  return line.item.modifierGroups
    .map((group) => {
      const optionSlug = line.selectedModifiers[group.slug];
      return group.options.find((option) => option.slug === optionSlug)?.name;
    })
    .filter((name): name is string => Boolean(name))
    .join(", ");
}
