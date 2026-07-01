import { formatRupiah } from "../../lib/format";
import type { CatalogCategory, CatalogItem, CatalogSort, QuickFilter } from "./catalogView";
import { menuItemBadges, menuItemImageSrc } from "./cashierItemView";
import type { MenuItem } from "./types";

type MenuCatalogPanelProps = {
  activeCategorySlug: string;
  catalogCategories: CatalogCategory[];
  catalogItems: CatalogItem[];
  catalogSort: CatalogSort;
  quickFilters: QuickFilter[];
  selectedItemSlug: string | undefined;
  onCategoryChange: (slug: string) => void;
  onItemClick: (item: MenuItem) => void;
  onQuickFilterToggle: (filter: QuickFilter) => void;
  onSortChange: (sort: CatalogSort) => void;
};

const quickFilterOptions: { label: string; value: QuickFilter }[] = [
  { label: "🔥 Best Seller", value: "bestSeller" },
  { label: "❄️ Iced", value: "iced" },
  { label: "◇ Low Sugar", value: "lowSugar" },
  { label: "✨ New Arrival", value: "newArrival" }
];

export function MenuCatalogPanel({
  activeCategorySlug,
  catalogCategories,
  catalogItems,
  catalogSort,
  quickFilters,
  selectedItemSlug,
  onCategoryChange,
  onItemClick,
  onQuickFilterToggle,
  onSortChange
}: MenuCatalogPanelProps) {
  return (
    <section className="cashier-panel menu-panel" aria-labelledby="menu-title">
      <div className="catalog-toolbar">
        <div>
          <h3 id="menu-title">Menu</h3>
          <p>Choose an item from the catalog.</p>
        </div>
      </div>

      <div className="catalog-tabs" role="tablist" aria-label="Menu categories">
        {catalogCategories.map((category) => (
          <button
            aria-selected={activeCategorySlug === category.slug}
            className={activeCategorySlug === category.slug ? "catalog-tab catalog-tab--active" : "catalog-tab"}
            key={category.slug}
            onClick={() => onCategoryChange(category.slug)}
            role="tab"
            type="button"
          >
            {category.name}
          </button>
        ))}
      </div>

      <div className="catalog-filters" aria-label="Quick filters">
        <span className="catalog-filters__label">Quick Filters:</span>
        {quickFilterOptions.map((filter) => (
          <button
            aria-label={filter.value === "bestSeller" ? "Best Seller" : filter.label.replace(/^[^A-Za-z]+ /, "")}
            aria-pressed={quickFilters.includes(filter.value)}
            className={quickFilters.includes(filter.value) ? "catalog-filter catalog-filter--active" : "catalog-filter"}
            key={filter.value}
            onClick={() => onQuickFilterToggle(filter.value)}
            type="button"
          >
            {filter.label}
          </button>
        ))}

        <label className="catalog-sort">
          <span>Sort by:</span>
          <select
            aria-label="Sort menu"
            onChange={(event) => onSortChange((event.currentTarget as HTMLSelectElement).value as CatalogSort)}
            value={catalogSort}
          >
            <option value="popular">Popular</option>
          </select>
        </label>
      </div>

      {catalogItems.length === 0 ? (
        <p className="catalog-empty" role="status">
          No menu items match the current filters.
        </p>
      ) : (
        <div className="menu-list menu-list--grid">
          {catalogItems.map(({ item }) => (
            <MenuCatalogCard
              item={item}
              isSelected={selectedItemSlug === item.slug}
              key={item.slug}
              onClick={() => onItemClick(item)}
            />
          ))}
        </div>
      )}
    </section>
  );
}

function MenuCatalogCard({
  item,
  isSelected,
  onClick
}: {
  item: MenuItem;
  isSelected: boolean;
  onClick: () => void;
}) {
  const badges = menuItemBadges(item);

  return (
    <button aria-pressed={isSelected} className="menu-item menu-card" onClick={onClick} type="button">
      <span className="menu-item__thumb" aria-hidden="true">
        <img alt="" src={menuItemImageSrc(item)} />
      </span>
      <span className="menu-item__content">
        {badges.length > 0 ? (
          <span className="menu-card__badges" aria-hidden="true">
            {badges.map((badge) => (
              <span className="menu-card__badge" key={badge}>
                {badge}
              </span>
            ))}
          </span>
        ) : null}
        <span>{item.name}</span>
        <span>{formatRupiah(item.priceRp)}</span>
      </span>
    </button>
  );
}
