-- GetTileHolder finds the last mine that took a tile as it finds a virus's: a GIN of its payloads, apart from the
-- others', so no index of the table is built again.
CREATE INDEX IF NOT EXISTS ledger_events_tile_mine ON ledger_events USING gin (payload jsonb_path_ops)
    WHERE kind = 'mine';
