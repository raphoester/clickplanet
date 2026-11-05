-- A snapshot a day of the numbers of every post of the clips job, sent by `clips stats`.
-- summary: the clips.v1.StatsSnapshot as protojson writes it, counts and states.
CREATE TABLE snapshots (
    taken_at timestamptz PRIMARY KEY,
    summary  jsonb       NOT NULL
);

-- One platform of one post at one snapshot. document: the row as the job wrote it, what Zernio answered as it came.
CREATE TABLE snapshot_rows (
    taken_at timestamptz NOT NULL REFERENCES snapshots (taken_at) ON DELETE CASCADE,
    post_id  text        NOT NULL,
    platform text        NOT NULL,
    document jsonb       NOT NULL,
    PRIMARY KEY (taken_at, post_id, platform)
);
