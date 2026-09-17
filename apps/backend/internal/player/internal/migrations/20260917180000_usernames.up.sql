-- A name is now a username: 3 to 20 characters, each an ASCII letter, a digit or an underscore, never starting
-- with "guest_" in any case, and unique ignoring case.
--
-- No client called SetName before usernames existed, so a row these deletes remove was not written by the game.
DELETE FROM profiles
WHERE name !~ '^[A-Za-z0-9_]{3,20}$' OR left(lower(name), 6) = 'guest_';

-- Of two names that differ only in case, the one saved first stays.
DELETE FROM profiles AS later
USING profiles AS earlier
WHERE lower(later.name) = lower(earlier.name)
  AND (earlier.updated_at, earlier.account_id) < (later.updated_at, later.account_id);

ALTER TABLE profiles DROP CONSTRAINT profiles_name_check;
ALTER TABLE profiles ADD CONSTRAINT profiles_name_check
    CHECK (name ~ '^[A-Za-z0-9_]{3,20}$' AND left(lower(name), 6) <> 'guest_');

-- The store maps a violation of this index to players.ErrNameTaken, by its name.
CREATE UNIQUE INDEX profiles_name_key ON profiles (lower(name));
