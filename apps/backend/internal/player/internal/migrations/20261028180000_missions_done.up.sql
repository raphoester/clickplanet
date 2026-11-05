ALTER TABLE stats ADD COLUMN missions_done bigint NOT NULL DEFAULT 0 CHECK (missions_done >= 0);
