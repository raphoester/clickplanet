-- The cut names stay cut, and the names only the new rule allows are deleted: the old CHECK refuses them.
-- What is left is ASCII, so its name_folded was lower(name) and the old index finds no two alike.
DROP INDEX profiles_name_key;
ALTER TABLE profiles DROP CONSTRAINT profiles_name_folded_check;
ALTER TABLE profiles DROP CONSTRAINT profiles_name_check;

DELETE FROM profiles WHERE name !~ '^[A-Za-z0-9_]{3,20}$';

ALTER TABLE profiles DROP COLUMN name_folded;
ALTER TABLE profiles ADD CONSTRAINT profiles_name_check
    CHECK (name ~ '^[A-Za-z0-9_]{3,20}$' AND left(lower(name), 6) <> 'guest_');
CREATE UNIQUE INDEX profiles_name_key ON profiles (lower(name));
