DELETE FROM profiles AS cut
WHERE char_length(cut.name) > 15
  AND EXISTS (
    SELECT 1 FROM profiles AS other
    WHERE other.account_id <> cut.account_id
      AND translate(left(other.name, 15), 'ABCDEFGHIJKLMNOPQRSTUVWXYZ', 'abcdefghijklmnopqrstuvwxyz')
        = translate(left(cut.name, 15), 'ABCDEFGHIJKLMNOPQRSTUVWXYZ', 'abcdefghijklmnopqrstuvwxyz')
      AND (char_length(other.name) <= 15
        OR (other.updated_at, other.account_id) < (cut.updated_at, cut.account_id)));

ALTER TABLE profiles DROP CONSTRAINT profiles_name_check;

UPDATE profiles SET name = left(name, 15) WHERE char_length(name) > 15;

ALTER TABLE profiles ADD COLUMN name_folded text;
UPDATE profiles SET name_folded = translate(name, 'ABCDEFGHIJKLMNOPQRSTUVWXYZ', 'abcdefghijklmnopqrstuvwxyz');
ALTER TABLE profiles ALTER COLUMN name_folded SET NOT NULL;

ALTER TABLE profiles ADD CONSTRAINT profiles_name_check CHECK (
    char_length(name) BETWEEN 3 AND 15
    AND name IS NFC NORMALIZED
    AND name = btrim(name, ' ')
    AND strpos(name, '  ') = 0
    AND name !~ '[\x01-\x1F\x21-\x2F\x3A-\x40\x5B-\x5E\x60\x7B-\x9F­؜᠎​-‏‪-‮⁠-⁯﻿]'
);
ALTER TABLE profiles ADD CONSTRAINT profiles_name_folded_check CHECK (
    name_folded <> '' AND left(name_folded, 6) <> 'guest_'
);

DROP INDEX profiles_name_key;
CREATE UNIQUE INDEX profiles_name_key ON profiles (name_folded);
