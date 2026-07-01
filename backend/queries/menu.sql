-- name: UpsertMenuCategory :one
insert into menu_categories (name, slug, sort_order, updated_at)
values ($1, $2, $3, now())
on conflict (slug) do update set
	name = excluded.name,
	sort_order = excluded.sort_order,
	updated_at = now()
returning id;

-- name: UpsertMenuItem :one
insert into menu_items (
	category_id,
	name,
	slug,
	price_rp,
	active,
	sort_order,
	image_path,
	popularity_rank,
	best_seller,
	promo,
	iced,
	low_sugar,
	new_arrival,
	updated_at
)
values ($1, $2, $3, $4, $5, $6, sqlc.narg(image_path), sqlc.narg(popularity_rank), $7, $8, $9, $10, $11, now())
on conflict (category_id, slug) do update set
	name = excluded.name,
	price_rp = excluded.price_rp,
	active = excluded.active,
	sort_order = excluded.sort_order,
	image_path = excluded.image_path,
	popularity_rank = excluded.popularity_rank,
	best_seller = excluded.best_seller,
	promo = excluded.promo,
	iced = excluded.iced,
	low_sugar = excluded.low_sugar,
	new_arrival = excluded.new_arrival,
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
select
	id,
	category_id,
	name,
	slug,
	price_rp,
	active,
	sort_order,
	image_path,
	popularity_rank,
	best_seller,
	promo,
	iced,
	low_sugar,
	new_arrival
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

-- name: ListCashierMenuRows :many
select
	c.id as category_id,
	c.name as category_name,
	c.slug as category_slug,
	i.id as item_id,
	i.name as item_name,
	i.slug as item_slug,
	i.price_rp,
	i.image_path,
	i.popularity_rank,
	i.best_seller,
	i.promo,
	i.iced,
	i.low_sugar,
	i.new_arrival,
	g.id as group_id,
	g.name as group_name,
	g.slug as group_slug,
	g.required,
	g.selection_type,
	o.id as option_id,
	o.name as option_name,
	o.slug as option_slug,
	o.price_delta_rp
from menu_categories c
join menu_items i on i.category_id = c.id and i.active = true
left join menu_item_modifier_groups mig on mig.menu_item_id = i.id
left join modifier_groups g on g.id = mig.modifier_group_id
left join modifier_options o on o.modifier_group_id = g.id
order by c.sort_order, c.id, i.sort_order, i.id, mig.sort_order, g.sort_order, g.id, o.sort_order, o.id;
