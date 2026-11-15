-- The UTC days with at least one take, counted live from here on. The days before are counted once by the operator's
-- progression.v1.AdminService/BackfillDaysPlayed, from planet's ledger.
ALTER TABLE stats ADD COLUMN IF NOT EXISTS days_played bigint NOT NULL DEFAULT 0 CHECK (days_played >= 0);
