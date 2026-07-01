import { describe, expect, it } from "vitest";
import type { CashierMenu } from "./types";
import { buildCatalogCategories, buildCatalogItems } from "./catalogView";

const menu: CashierMenu = {
  categories: [
    {
      name: "Coffee",
      slug: "coffee",
      items: [
        {
          name: "Latte",
          slug: "latte",
          priceRp: 25000,
          popularityRank: 20,
          bestSeller: true,
          iced: true,
          lowSugar: true,
          modifierGroups: []
        },
        {
          name: "Americano",
          slug: "americano",
          priceRp: 18000,
          popularityRank: 10,
          bestSeller: true,
          modifierGroups: []
        }
      ]
    },
    {
      name: "Tea",
      slug: "tea",
      items: [
        {
          name: "Iced Tea",
          slug: "iced-tea",
          priceRp: 15000,
          iced: true,
          modifierGroups: []
        }
      ]
    },
    {
      name: "Snacks",
      slug: "snacks",
      items: [
        {
          name: "Muffin",
          slug: "muffin",
          priceRp: 20000,
          newArrival: true,
          modifierGroups: []
        }
      ]
    }
  ]
};

describe("catalog view helpers", () => {
  it("returns All plus backend categories in display order", () => {
    expect(buildCatalogCategories(menu)).toEqual([
      { name: "All", slug: "all" },
      { name: "Coffee", slug: "coffee" },
      { name: "Tea", slug: "tea" },
      { name: "Snacks", slug: "snacks" }
    ]);
  });

  it("searches item names case-insensitively without mutating the menu", () => {
    const before = structuredClone(menu);

    expect(buildCatalogItems(menu, { searchQuery: "LAT", categorySlug: "all", quickFilters: [], sort: "popular" }).map((entry) => entry.item.slug)).toEqual([
      "latte"
    ]);
    expect(menu).toEqual(before);
  });

  it("filters by category including All", () => {
    expect(buildCatalogItems(menu, { searchQuery: "", categorySlug: "tea", quickFilters: [], sort: "popular" }).map((entry) => entry.item.slug)).toEqual([
      "iced-tea"
    ]);
    expect(buildCatalogItems(menu, { searchQuery: "", categorySlug: "all", quickFilters: [], sort: "popular" })).toHaveLength(4);
  });

  it("applies quick filters from backend metadata", () => {
    expect(buildCatalogItems(menu, { searchQuery: "", categorySlug: "all", quickFilters: ["bestSeller"], sort: "popular" }).map((entry) => entry.item.slug)).toEqual([
      "americano",
      "latte"
    ]);
    expect(buildCatalogItems(menu, { searchQuery: "", categorySlug: "all", quickFilters: ["newArrival"], sort: "popular" }).map((entry) => entry.item.slug)).toEqual([
      "muffin"
    ]);
  });

  it("sorts by popularity rank and falls back to backend order", () => {
    expect(buildCatalogItems(menu, { searchQuery: "", categorySlug: "all", quickFilters: ["iced"], sort: "popular" }).map((entry) => entry.item.slug)).toEqual([
      "latte",
      "iced-tea"
    ]);
  });
});
