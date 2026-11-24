DROP TABLE rejections;
DROP TYPE rejection_reason;
DROP TABLE submissions;
DROP TYPE submission_state;
DROP TYPE submission_kind;

ALTER TABLE flags
    DROP COLUMN pin_latitude,
    DROP COLUMN pin_longitude;
