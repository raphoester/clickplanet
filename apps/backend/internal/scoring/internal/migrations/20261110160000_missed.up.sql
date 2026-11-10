ALTER TYPE outbox_event ADD VALUE IF NOT EXISTS 'scoring.v1.RoundMissed';
ALTER TYPE outbox_event ADD VALUE IF NOT EXISTS 'scoring.v1.BookMissed';
