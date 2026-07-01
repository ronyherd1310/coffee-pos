alter table menu_items
	add column if not exists image_path text check (image_path is null or btrim(image_path) <> ''),
	add column if not exists popularity_rank integer check (popularity_rank is null or popularity_rank >= 0),
	add column if not exists best_seller boolean not null default false,
	add column if not exists promo boolean not null default false,
	add column if not exists iced boolean not null default false,
	add column if not exists low_sugar boolean not null default false,
	add column if not exists new_arrival boolean not null default false;
