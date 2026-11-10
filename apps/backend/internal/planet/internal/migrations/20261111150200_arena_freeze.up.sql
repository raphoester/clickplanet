-- Each arena's freeze, read from its battle's rule set as it opened: the set plays only once the arena opened.
ALTER TABLE arenas ADD COLUMN freeze_ms bigint NOT NULL DEFAULT 8000 CHECK (freeze_ms >= 0);
ALTER TABLE arenas ALTER COLUMN freeze_ms DROP DEFAULT;
