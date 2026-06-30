-- name: UpsertMenuCategory :one
insert into menu_categories (name, slug, sort_order, updated_at)
values ($1, $2, $3, now())
on conflict (slug) do update set
	name = excluded.name,
	sort_order = excluded.sort_order,
	updated_at = now()
returning id;

-- name: UpsertMenuItem :one
insert into menu_items (category_id, name, slug, price_rp, active, sort_order, updated_at)
values ($1, $2, $3, $4, $5, $6, now())
on conflict (category_id, slug) do update set
	name = excluded.name,
	price_rp = excluded.price_rp,
	active = excluded.active,
	sort_order = excluded.sort_order,
	updated_at = now()
returning id;

-- name: UpsertModifierGroup :one
insert into modifier_groups (name, slug, required, selection_type, sort_order, updated_at)
values ($1, $2, $3, $4, $5, now())
on conflict (slug) do update set
	name = excluded.name,
	required = excluded.required,
	selection_type = excluded.selection_type,
	sort_order = excluded.sort_order,
	updated_at = now()
returning id;

-- name: UpsertMenuItemModifierGroup :exec
insert into menu_item_modifier_groups (menu_item_id, modifier_group_id, sort_order)
values ($1, $2, $3)
on conflict (menu_item_id, modifier_group_id) do update set
	sort_order = excluded.sort_order;

-- name: UpsertModifierOption :one
insert into modifier_options (modifier_group_id, name, slug, price_delta_rp, sort_order, updated_at)
values ($1, $2, $3, $4, $5, now())
on conflict (modifier_group_id, slug) do update set
	name = excluded.name,
	price_delta_rp = excluded.price_delta_rp,
	sort_order = excluded.sort_order,
	updated_at = now()
returning id;

-- name: ListMenuCategories :many
select id, name, slug, sort_order
from menu_categories
order by sort_order, id;

-- name: ListMenuItems :many
select id, category_id, name, slug, price_rp, active, sort_order
from menu_items
order by sort_order, id;

-- name: ListModifierGroups :many
select id, name, slug, required, selection_type, sort_order
from modifier_groups
order by sort_order, id;

-- name: ListModifierOptions :many
select id, modifier_group_id, name, slug, price_delta_rp, sort_order
from modifier_options
order by sort_order, id;

-- name: CountMenuItemModifierGroups :one
select count(*) from menu_item_modifier_groups;
