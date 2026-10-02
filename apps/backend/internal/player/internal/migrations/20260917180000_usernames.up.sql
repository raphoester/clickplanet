DELETE FROM profiles
WHERE name !~ '^[A-Za-z0-9_]{3,20}$' OR left(lower(name), 6) = 'guest_';

DELETE FROM profiles AS later
USING profiles AS earlier
WHERE lower(later.name) = lower(earlier.name)
  AND (earlier.updated_at, earlier.account_id) < (later.updated_at, later.account_id);

ALTER TABLE profiles DROP CONSTRAINT profiles_name_check;
ALTER TABLE profiles ADD CONSTRAINT profiles_name_check
    CHECK (name ~ '^[A-Za-z0-9_]{3,20}$' AND left(lower(name), 6) <> 'guest_');

CREATE UNIQUE INDEX profiles_name_key ON profiles (lower(name));
