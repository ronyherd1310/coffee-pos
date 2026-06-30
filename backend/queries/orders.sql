-- name: AllocateDailyQueueNumber :one
insert into daily_queue_counters (business_date, last_queue_number, updated_at)
values ($1, 1, now())
on conflict (business_date) do update set
	last_queue_number = daily_queue_counters.last_queue_number + 1,
	updated_at = now()
returning last_queue_number;

-- name: CreateOrder :one
insert into orders (
	business_date,
	queue_number,
	status,
	payment_method,
	paid_at,
	cancelled_at,
	note,
	total_rp,
	client_request_id,
	request_hash,
	updated_at
)
values ($1, $2, 'paid', $3, $4, null, $5, $6, $7, $8, now())
returning id;

-- name: GetOrderIdempotency :one
select id, request_hash
from orders
where client_request_id = $1;

-- name: CreateOrderLine :one
insert into order_lines (
	order_id,
	menu_item_id,
	menu_item_slug,
	menu_item_name,
	unit_price_rp,
	quantity,
	line_total_rp,
	display_order
)
values ($1, $2, $3, $4, $5, $6, $7, $8)
returning id;

-- name: CreateOrderLineModifier :exec
insert into order_line_modifiers (
	order_line_id,
	modifier_group_id,
	modifier_option_id,
	group_slug,
	group_name,
	option_slug,
	option_name,
	price_delta_rp,
	display_order
)
values ($1, $2, $3, $4, $5, $6, $7, $8, $9);

-- name: GetPaidOrder :many
select
	o.id as order_id,
	o.business_date,
	o.queue_number,
	o.status,
	o.payment_method,
	o.paid_at,
	o.cancelled_at,
	o.note,
	o.total_rp,
	ol.id as order_line_id,
	ol.menu_item_slug,
	ol.menu_item_name,
	ol.unit_price_rp,
	ol.quantity,
	ol.line_total_rp,
	ol.display_order as line_display_order,
	olm.group_slug,
	olm.group_name,
	olm.option_slug,
	olm.option_name,
	olm.price_delta_rp,
	olm.display_order as modifier_display_order
from orders o
join order_lines ol on ol.order_id = o.id
left join order_line_modifiers olm on olm.order_line_id = ol.id
where o.id = $1
order by ol.display_order, ol.id, olm.display_order, olm.id;

-- name: CancelOrder :execrows
update orders
set status = 'cancelled',
	cancelled_at = $2,
	updated_at = now()
where id = $1
	and business_date = $3
	and status = 'paid';
