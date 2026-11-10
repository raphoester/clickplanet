-- How a round orders its countries: every round kept was scored on its gain.
CREATE TYPE round_ranking AS ENUM ('gain', 'held_and_gain');
ALTER TABLE rounds ADD COLUMN ranking round_ranking NOT NULL DEFAULT 'gain';
ALTER TABLE rounds ALTER COLUMN ranking DROP DEFAULT;
