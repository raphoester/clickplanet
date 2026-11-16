-- GetTileHolder finds the last virus that took a tile as it finds a spread's: a GIN of its payloads, apart from the
-- spreads', encloses' and bombs', so no index of the table is built again.
CREATE INDEX IF NOT EXISTS ledger_events_tile_virus ON ledger_events USING gin (payload jsonb_path_ops)
    WHERE kind = 'virus';
