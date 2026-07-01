import { formatRupiah } from "../../lib/format";
import { hasRequiredModifiers, menuItemImageSrc, optionControlClass } from "./cashierItemView";
import type { SelectedModifiers } from "./orderDraft";
import type { MenuItem } from "./types";

type SelectedItemPanelProps = {
  item: MenuItem;
  quantity: number;
  selectedModifiers: SelectedModifiers;
  onModifierChange: (groupSlug: string, optionSlug: string) => void;
  onQuantityChange: (quantity: number) => void;
  onCancel: () => void;
  onAddLine: () => void;
};

export function SelectedItemPanel({
  item,
  quantity,
  selectedModifiers,
  onModifierChange,
  onQuantityChange,
  onCancel,
  onAddLine
}: SelectedItemPanelProps) {
  const canAdd = hasRequiredModifiers(item, selectedModifiers);

  return (
    <section className="cashier-panel selected-item-panel" aria-labelledby="selected-item-title">
      <div className="selected-item-panel__title">
        <span className="menu-item__thumb menu-item__thumb--small" aria-hidden="true">
          <img alt="" src={menuItemImageSrc(item)} />
        </span>
        <div>
          <h3 id="selected-item-title" aria-label={`Configure ${item.name}`}>
            {item.name}
          </h3>
          <p>{formatRupiah(item.priceRp)}</p>
        </div>
      </div>

      {item.modifierGroups.map((group) => (
        <fieldset className="modifier-group" key={group.slug}>
          <legend>
            {group.name}
            {group.required ? <span>Required</span> : null}
          </legend>
          <div className="option-grid">
            {group.options.map((option) => {
              const label =
                option.priceDeltaRp > 0
                  ? `${option.name} +${formatRupiah(option.priceDeltaRp)}`
                  : option.name;
              const id = `modifier-${item.slug}-${group.slug}-${option.slug}`;

              return (
                <label className={optionControlClass(option.slug)} htmlFor={id} key={option.slug}>
                  <input
                    checked={selectedModifiers[group.slug] === option.slug}
                    id={id}
                    name={`modifier-${item.slug}-${group.slug}`}
                    onChange={() => onModifierChange(group.slug, option.slug)}
                    type="radio"
                  />
                  <span className="option-control__label">{label}</span>
                </label>
              );
            })}
          </div>
        </fieldset>
      ))}

      <div className="quantity-row">
        <span>Quantity</span>
        <div className="quantity-stepper">
          <button
            aria-label="Decrease selected item quantity"
            className="stepper-button"
            onClick={() => onQuantityChange(quantity - 1)}
            type="button"
          >
            -
          </button>
          <input aria-label="Selected item quantity" readOnly type="number" value={quantity} />
          <button
            aria-label="Increase selected item quantity"
            className="stepper-button"
            onClick={() => onQuantityChange(quantity + 1)}
            type="button"
          >
            +
          </button>
        </div>
      </div>

      <div className="selected-item-panel__actions">
        <button className="button button--secondary" onClick={onCancel} type="button">
          <span>Cancel customization</span>
        </button>
        <button className="button button--primary button--add-item" disabled={!canAdd} onClick={onAddLine} type="button">
          <span>Add Item To Order</span>
        </button>
      </div>
    </section>
  );
}
