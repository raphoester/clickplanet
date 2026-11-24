-- Every flag opened since the game shipped starts somewhere in Capital Siege. Production held a test flag of Corsica
-- and Brittany, which left the flags the game ships with, when this ran.
ALTER TABLE flags
    ADD COLUMN pin_latitude  double precision,
    ADD COLUMN pin_longitude double precision;

UPDATE flags SET pin_latitude = 41.9192, pin_longitude = 8.7386 WHERE lower(name) = 'corsica';
UPDATE flags SET pin_latitude = 48.1173, pin_longitude = -1.6778 WHERE lower(name) = 'brittany';

DO $$
BEGIN
    IF EXISTS (SELECT 1 FROM flags WHERE pin_latitude IS NULL) THEN
        RAISE EXCEPTION 'a flag opened since the game shipped has no pin: give it one in this migration';
    END IF;
END
$$;

ALTER TABLE flags
    ALTER COLUMN pin_latitude SET NOT NULL,
    ALTER COLUMN pin_longitude SET NOT NULL;

-- A flag a player submitted for review. Its picture is drawn again and kept private until its approval.
CREATE TYPE submission_kind AS ENUM ('new', 'change');
CREATE TYPE submission_state AS ENUM ('waiting', 'approved', 'rejected', 'opened');

CREATE TABLE submissions (
    id              uuid             PRIMARY KEY,
    -- Its founder. A deleted account's id stays, as on the ledger's takes: it names nobody once purged.
    account_id      uuid             NOT NULL,
    kind            submission_kind  NOT NULL,
    -- The flag a change is for, or the flag opened from a new one; empty until then.
    flag            text             NOT NULL,
    name            text             NOT NULL,
    pin_latitude    double precision NOT NULL,
    pin_longitude   double precision NOT NULL,
    picture_hash    text             NOT NULL,
    picture_width   integer          NOT NULL,
    picture_height  integer          NOT NULL,
    picture_stretch boolean          NOT NULL,
    picture_focus_x real             NOT NULL,
    picture_focus_y real             NOT NULL,
    -- The share of it, in percent, in colours other than its 8 most common.
    picture_other_colors integer     NOT NULL,
    -- False once a rejected picture was deleted.
    picture_held    boolean          NOT NULL,
    state           submission_state NOT NULL,
    submitted_at    timestamptz      NOT NULL,
    -- When it took its state.
    decided_at      timestamptz      NOT NULL
);

CREATE INDEX submissions_account ON submissions (account_id);

CREATE INDEX submissions_waiting ON submissions (submitted_at) WHERE state = 'waiting';

CREATE INDEX submissions_decided ON submissions (decided_at) WHERE state <> 'waiting';

CREATE INDEX submissions_rejected_held ON submissions (decided_at) WHERE state = 'rejected' AND picture_held;

CREATE TYPE rejection_reason AS ENUM ('not_a_place', 'exists', 'unreadable', 'hateful', 'other');

-- Why a rejected submission was refused, which its player reads.
CREATE TABLE rejections (
    submission_id uuid             PRIMARY KEY REFERENCES submissions (id) ON DELETE CASCADE,
    reason        rejection_reason NOT NULL,
    note          text             NOT NULL
);
