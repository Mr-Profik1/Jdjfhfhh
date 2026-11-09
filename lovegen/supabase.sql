-- Выполни целиком в Supabase: SQL Editor -> New query -> вставь -> Run.

-- 1. Профили пользователей
create table if not exists public.profiles (
  id uuid primary key references auth.users(id) on delete cascade,
  email text,
  role text not null default 'user' check (role in ('user','admin')),
  banned boolean not null default false,
  created_at timestamptz not null default now()
);

-- 2. Генерации
create table if not exists public.generations (
  id bigint generated always as identity primary key,
  user_id uuid not null references public.profiles(id) on delete cascade,
  prompt text not null check (char_length(prompt) <= 1000),
  provider text,
  image_url text not null,
  created_at timestamptz not null default now()
);
create index if not exists generations_user_idx on public.generations(user_id, created_at desc);
create index if not exists generations_created_idx on public.generations(created_at desc);

-- 3. Профиль создаётся автоматически при регистрации
create or replace function public.handle_new_user() returns trigger
language plpgsql security definer set search_path = public as $$
begin
  insert into public.profiles(id, email) values (new.id, new.email) on conflict do nothing;
  return new;
end $$;
drop trigger if exists on_auth_user_created on auth.users;
create trigger on_auth_user_created after insert on auth.users
  for each row execute function public.handle_new_user();

-- 4. Вспомогательные функции
create or replace function public.is_admin() returns boolean
language sql security definer stable set search_path = public as $$
  select exists(select 1 from public.profiles where id = auth.uid() and role = 'admin');
$$;
create or replace function public.is_banned() returns boolean
language sql security definer stable set search_path = public as $$
  select coalesce((select banned from public.profiles where id = auth.uid()), false);
$$;

-- 5. Защита строк (RLS)
alter table public.profiles enable row level security;
alter table public.generations enable row level security;

drop policy if exists "profiles read" on public.profiles;
create policy "profiles read" on public.profiles for select
  using (id = auth.uid() or public.is_admin());

drop policy if exists "profiles admin update" on public.profiles;
create policy "profiles admin update" on public.profiles for update
  using (public.is_admin()) with check (public.is_admin());

drop policy if exists "gen read" on public.generations;
create policy "gen read" on public.generations for select
  using (user_id = auth.uid() or public.is_admin());

drop policy if exists "gen insert" on public.generations;
create policy "gen insert" on public.generations for insert
  with check (user_id = auth.uid() and not public.is_banned());

drop policy if exists "gen delete" on public.generations;
create policy "gen delete" on public.generations for delete
  using (user_id = auth.uid() or public.is_admin());

-- 6. Хранилище картинок (для режима Hugging Face)
insert into storage.buckets (id, name, public) values ('images', 'images', true)
  on conflict (id) do nothing;

drop policy if exists "images upload own folder" on storage.objects;
create policy "images upload own folder" on storage.objects for insert to authenticated
  with check (bucket_id = 'images' and (storage.foldername(name))[1] = auth.uid()::text and not public.is_banned());

drop policy if exists "images delete" on storage.objects;
create policy "images delete" on storage.objects for delete to authenticated
  using (bucket_id = 'images' and ((storage.foldername(name))[1] = auth.uid()::text or public.is_admin()));

-- 7. СДЕЛАТЬ СЕБЯ АДМИНОМ: сначала зарегистрируйся на сайте, потом выполни это,
--    подставив свою почту (раскомментируй строки):
-- update public.profiles set role = 'admin' where email = 'ТВОЯ_ПОЧТА@example.com';
