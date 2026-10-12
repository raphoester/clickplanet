CREATE TYPE identity_provider AS ENUM ('google', 'discord', 'email');

ALTER TABLE identities ALTER COLUMN provider TYPE identity_provider USING provider::identity_provider;
