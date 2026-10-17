CREATE TYPE account_role AS ENUM ('player', 'operator');

ALTER TABLE accounts ADD COLUMN role account_role NOT NULL DEFAULT 'player';
