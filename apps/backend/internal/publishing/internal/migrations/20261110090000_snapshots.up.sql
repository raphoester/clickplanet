-- A snapshot a day of the numbers of every post the game made, taken by the server since 2026-11-10.
-- summary: the publishing.v1.StatsSnapshot as protojson writes it, counts and states.
CREATE TABLE snapshots (
    taken_at timestamptz PRIMARY KEY,
    summary  jsonb       NOT NULL
);

-- One platform of one post at one snapshot. document: the row as the snapshot wrote it, what Zernio answered as it came.
CREATE TABLE snapshot_rows (
    taken_at timestamptz NOT NULL REFERENCES snapshots (taken_at) ON DELETE CASCADE,
    post_id  text        NOT NULL,
    platform text        NOT NULL,
    document jsonb       NOT NULL,
    PRIMARY KEY (taken_at, post_id, platform)
);

-- GetPostLinks: the newest address a snapshot read of a post on a platform.
CREATE INDEX snapshot_rows_post ON snapshot_rows (post_id, platform, taken_at DESC);

-- The snapshots the clips job sent the clips module until then, copied once: the summary is the same message as
-- clips.v1.StatsSnapshot was, and a row the same document. The clips tables are left as they were.
DO $$
BEGIN
    IF to_regclass('clips.snapshots') IS NOT NULL THEN
        INSERT INTO snapshots (taken_at, summary)
        SELECT taken_at, summary FROM clips.snapshots
        ON CONFLICT (taken_at) DO NOTHING;

        INSERT INTO snapshot_rows (taken_at, post_id, platform, document)
        SELECT taken_at, post_id, platform, document FROM clips.snapshot_rows
        ON CONFLICT (taken_at, post_id, platform) DO NOTHING;
    END IF;
END $$;
