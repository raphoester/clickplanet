-- IF NOT EXISTS: a run killed mid-build leaves the version dirty, and forcing it back runs this again.
CREATE INDEX IF NOT EXISTS ledger_events_tile_take ON ledger_events (tile, position) WHERE kind = 'take';

CREATE INDEX IF NOT EXISTS ledger_events_tile_payload ON ledger_events USING gin (payload jsonb_path_ops)
    WHERE kind IN ('spread', 'enclose', 'bomb');
