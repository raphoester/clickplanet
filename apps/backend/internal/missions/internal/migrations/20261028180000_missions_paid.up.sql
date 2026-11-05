ALTER TYPE outbox_event ADD VALUE IF NOT EXISTS 'missions.v1.MissionsPaid';

CREATE INDEX mission_payments_account ON mission_payments (account_id);
