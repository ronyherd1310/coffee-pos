create table if not exists menu_categories (
	id bigserial primary key,
	name text not null check (btrim(name) <> ''),
	slug text not null check (btrim(slug) <> ''),
	sort_order integer not null default 0,
	created_at timestamptz not null default now(),
	updated_at timestamptz not null default now(),
	unique (name),
	unique (slug)
);

create table if not exists menu_items (
	id bigserial primary key,
	category_id bigint not null references menu_categories(id) on delete restrict,
	name text not null check (btrim(name) <> ''),
	slug text not null check (btrim(slug) <> ''),
	price_rp integer not null check (price_rp >= 0),
	active boolean not null default true,
	sort_order integer not null default 0,
	created_at timestamptz not null default now(),
	updated_at timestamptz not null default now(),
	unique (category_id, name),
	unique (category_id, slug)
);

create table if not exists modifier_groups (
	id bigserial primary key,
	name text not null check (btrim(name) <> ''),
	slug text not null check (btrim(slug) <> ''),
	required boolean not null,
	selection_type text not null check (selection_type = 'single'),
	sort_order integer not null default 0,
	created_at timestamptz not null default now(),
	updated_at timestamptz not null default now(),
	unique (name),
	unique (slug)
);

create table if not exists menu_item_modifier_groups (
	menu_item_id bigint not null references menu_items(id) on delete cascade,
	modifier_group_id bigint not null references modifier_groups(id) on delete cascade,
	sort_order integer not null default 0,
	primary key (menu_item_id, modifier_group_id)
);

create table if not exists modifier_options (
	id bigserial primary key,
	modifier_group_id bigint not null references modifier_groups(id) on delete cascade,
	name text not null check (btrim(name) <> ''),
	slug text not null check (btrim(slug) <> ''),
	price_delta_rp integer not null check (price_delta_rp >= 0),
	sort_order integer not null default 0,
	created_at timestamptz not null default now(),
	updated_at timestamptz not null default now(),
	unique (modifier_group_id, name),
	unique (modifier_group_id, slug)
);
