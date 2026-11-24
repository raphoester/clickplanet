-- Each flag's pin as the latest fact that named one told it (flags.v1.FlagOpened, FlagMoved): where it starts an arena.
-- A row heard before this keeps none until the flags module's restatement at boot tells it.
ALTER TABLE heard_flags
    ADD COLUMN pin_latitude  double precision,
    ADD COLUMN pin_longitude double precision,
    ADD COLUMN pinned_at     timestamptz,
    ADD CONSTRAINT heard_flags_pin_whole CHECK (
        (pinned_at IS NULL) = (pin_latitude IS NULL) AND (pinned_at IS NULL) = (pin_longitude IS NULL)
    );
