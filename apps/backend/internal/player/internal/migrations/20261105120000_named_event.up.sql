-- Told whenever an account's username is set, so the alerts module gives the Discord member the same nickname.
ALTER TYPE outbox_event ADD VALUE IF NOT EXISTS 'player.v1.PlayerNamed';
