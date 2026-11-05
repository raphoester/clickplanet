ALTER TYPE outbox_event ADD VALUE IF NOT EXISTS 'player.v1.PlayerLeaving';
ALTER TYPE outbox_event ADD VALUE IF NOT EXISTS 'player.v1.PlayerReturned';
ALTER TYPE outbox_event ADD VALUE IF NOT EXISTS 'player.v1.PlayerForgotten';
