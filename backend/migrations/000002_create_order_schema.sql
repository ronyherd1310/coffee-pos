create table if not exists daily_queue_counters (
	business_date date primary key,
	last_queue_number integer not null check (last_queue_number >= 0),
	created_at timestamptz not null default now(),
	updated_at timestamptz not null default now()
);

create table if not exists orders (
	id bigserial primary key,
	business_date date not null,
	queue_number integer not null check (queue_number > 0),
	status text not null check (status in ('paid', 'cancelled')),
	payment_method text not null check (payment_method in ('cash', 'qris')),
	paid_at timestamptz not null,
	cancelled_at timestamptz,
	note text check (note is null or length(note) <= 500),
	total_rp bigint not null check (total_rp >= 0),
	client_request_id uuid not null,
	request_hash bytea not null,
	created_at timestamptz not null default now(),
	updated_at timestamptz not null default now(),
	unique (business_date, queue_number),
	unique (client_request_id),
	check ((status = 'paid' and cancelled_at is null) or (status = 'cancelled' and cancelled_at is not null))
);

create table if not exists order_lines (
	id bigserial primary key,
	order_id bigint not null references orders(id) on delete restrict,
	menu_item_id bigint not null references menu_items(id) on delete restrict,
	menu_item_slug text not null check (btrim(menu_item_slug) <> ''),
	menu_item_name text not null check (btrim(menu_item_name) <> ''),
	unit_price_rp bigint not null check (unit_price_rp >= 0),
	quantity integer not null check (quantity between 1 and 99),
	line_total_rp bigint not null check (line_total_rp >= 0),
	display_order integer not null check (display_order >= 0),
	created_at timestamptz not null default now(),
	unique (order_id, display_order)
);

create table if not exists order_line_modifiers (
	id bigserial primary key,
	order_line_id bigint not null references order_lines(id) on delete restrict,
	modifier_group_id bigint not null references modifier_groups(id) on delete restrict,
	modifier_option_id bigint not null references modifier_options(id) on delete restrict,
	group_slug text not null check (btrim(group_slug) <> ''),
	group_name text not null check (btrim(group_name) <> ''),
	option_slug text not null check (btrim(option_slug) <> ''),
	option_name text not null check (btrim(option_name) <> ''),
	price_delta_rp bigint not null check (price_delta_rp >= 0),
	display_order integer not null check (display_order >= 0),
	created_at timestamptz not null default now(),
	unique (order_line_id, display_order),
	unique (order_line_id, group_slug)
);
