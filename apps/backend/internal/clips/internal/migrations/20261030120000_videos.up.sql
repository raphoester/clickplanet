-- The videos the operator posts by hand: the file is in the bucket under clips/videos/<video_id>.
CREATE TABLE videos (
    video_id   uuid        PRIMARY KEY,
    name       text        NOT NULL CHECK (name <> ''),
    caption    text        NOT NULL,
    media_type text        NOT NULL,
    size_bytes bigint      NOT NULL CHECK (size_bytes > 0),
    kept_at    timestamptz NOT NULL
);

CREATE INDEX videos_kept_at ON videos (kept_at DESC);
