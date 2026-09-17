-- The deleted profiles do not come back.
DROP INDEX profiles_name_key;

ALTER TABLE profiles DROP CONSTRAINT profiles_name_check;
ALTER TABLE profiles ADD CONSTRAINT profiles_name_check CHECK (char_length(name) BETWEEN 1 AND 24);
