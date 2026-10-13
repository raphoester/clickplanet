CREATE TYPE media_type AS ENUM (
    'image/jpeg', 'image/png', 'image/webp', 'image/gif', 'image/heic', 'image/heif',
    'video/mp4', 'video/quicktime', 'video/webm'
);

CREATE TABLE attachments (
    note_id    uuid       NOT NULL REFERENCES notes (id),
    position   smallint   NOT NULL,
    media_type media_type NOT NULL,
    size_bytes bigint     NOT NULL,
    PRIMARY KEY (note_id, position)
);
