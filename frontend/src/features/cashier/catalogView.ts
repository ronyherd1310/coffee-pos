import type { CashierMenu, MenuCategory, MenuItem } from "./types";

export type CatalogCategory = Pick<MenuCategory, "name" | "slug">;

export type QuickFilter = "bestSeller" | "iced" | "lowSugar" | "newArrival";

export type CatalogSort = "popular";

export type CatalogViewState = {
  searchQuery: string;
  categorySlug: string;
  quickFilters: QuickFilter[];
  sort: CatalogSort;
};

export type CatalogItem = {
  item: MenuItem;
  categoryName: string;
  categorySlug: string;
  backendIndex: number;
};

export function buildCatalogCategories(menu: CashierMenu): CatalogCategory[] {
  return [{ name: "All", slug: "all" }, ...menu.categories.map(({ name, slug }) => ({ name, slug }))];
}

export function buildCatalogItems(menu: CashierMenu, state: CatalogViewState): CatalogItem[] {
  const query = state.searchQuery.trim().toLowerCase();
  const flattened = flattenCatalog(menu);

  const filtered = flattened.filter((entry) => {
    if (state.categorySlug !== "all" && entry.categorySlug !== state.categorySlug) {
      return false;
    }
    if (query && !entry.item.name.toLowerCase().includes(query)) {
      return false;
    }
    return state.quickFilters.every((filter) => entry.item[filter] === true);
  });

  if (state.sort === "popular") {
    return [...filtered].sort(comparePopular);
  }

  return filtered;
}

function flattenCatalog(menu: CashierMenu): CatalogItem[] {
  let backendIndex = 0;
  return menu.categories.flatMap((category) =>
    category.items.map((item) => ({
      backendIndex: backendIndex++,
      categoryName: category.name,
      categorySlug: category.slug,
      item
    }))
  );
}

function comparePopular(left: CatalogItem, right: CatalogItem): number {
  const leftRank = left.item.popularityRank ?? Number.MAX_SAFE_INTEGER;
  const rightRank = right.item.popularityRank ?? Number.MAX_SAFE_INTEGER;

  if (leftRank !== rightRank) {
    return leftRank - rightRank;
  }

  return left.backendIndex - right.backendIndex;
}
