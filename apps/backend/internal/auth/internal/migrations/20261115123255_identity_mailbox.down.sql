-- An emailed code's subject is its address again: the keepers first, since one holds a mailbox an identity given up
-- had as its address. Fails when a sign-in after the up linked a form of an address an identity given up holds: two
-- identities of one address, as before the up.
UPDATE identities SET subject = lower(email)
WHERE provider = 'email' AND subject NOT LIKE 'alias:%' AND subject <> lower(email);

UPDATE identities SET subject = regexp_replace(subject, '^alias:[^:]*:', '')
WHERE provider = 'email' AND subject LIKE 'alias:%';

DROP INDEX identities_mailbox;
CREATE INDEX identities_email ON identities (lower(email)) WHERE email_verified;

ALTER TABLE identities DROP COLUMN mailbox;
