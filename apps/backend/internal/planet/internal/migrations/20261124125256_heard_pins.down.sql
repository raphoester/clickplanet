ALTER TABLE heard_flags
    DROP CONSTRAINT heard_flags_pin_whole,
    DROP COLUMN pin_latitude,
    DROP COLUMN pin_longitude,
    DROP COLUMN pinned_at;
