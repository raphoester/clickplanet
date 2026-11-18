-- Told when a player leaves a username for another, so player keeps it among the account's past names.
ALTER TYPE outbox_event ADD VALUE IF NOT EXISTS 'player.v1.NameLeft';
