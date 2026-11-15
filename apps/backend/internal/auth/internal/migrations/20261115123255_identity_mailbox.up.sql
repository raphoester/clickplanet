-- The mailbox an identity's verified address delivers to (accounts.MailboxOf), which "one address is one player" now
-- compares, and the subject of an emailed code's identity. Go writes it from now on; this folds the rows already kept,
-- once, with the same rules: the migration's test runs both over the same addresses.
ALTER TABLE identities ADD COLUMN mailbox text;

CREATE FUNCTION pg_temp.mailbox_of(address text) RETURNS text LANGUAGE sql IMMUTABLE AS $$
    SELECT CASE
        WHEN domain IS NULL OR domain NOT IN (
            'gmail.com', 'googlemail.com',
            'outlook.com', 'outlook.fr', 'outlook.de', 'outlook.es', 'outlook.it',
            'hotmail.com', 'hotmail.fr', 'hotmail.co.uk', 'hotmail.de', 'hotmail.it', 'hotmail.es',
            'live.com', 'live.fr', 'live.co.uk', 'msn.com',
            'icloud.com', 'me.com', 'mac.com',
            'proton.me', 'protonmail.com', 'protonmail.ch', 'pm.me',
            'fastmail.com', 'fastmail.fm',
            'yandex.ru', 'yandex.com', 'ya.ru'
        ) THEN lowered
        WHEN domain IN ('gmail.com', 'googlemail.com') AND replace(split_part(name, '+', 1), '.', '') <> ''
            THEN replace(split_part(name, '+', 1), '.', '') || '@gmail.com'
        WHEN domain NOT IN ('gmail.com', 'googlemail.com') AND split_part(name, '+', 1) <> ''
            THEN split_part(name, '+', 1) || '@' || domain
        ELSE lowered
    END
    FROM (
        SELECT lowered, substring(lowered FROM '^(.*)@[^@]*$') AS name, substring(lowered FROM '@([^@]*)$') AS domain
        FROM (SELECT lower(btrim(address)) AS lowered) AS trimmed
    ) AS parts
$$;

UPDATE identities SET mailbox = pg_temp.mailbox_of(email) WHERE email_verified;

-- Each mailbox keeps one emailed code's identity: the oldest on the account that owns it, the account of its oldest
-- verified identity, as AccountOfMailbox reads it. Any other is one a form of the address made beside it: it gives the
-- subject up and keeps its address, so no sign-in finds it again and the operator's search still does.
WITH owners AS (
    SELECT DISTINCT ON (mailbox) mailbox, account_id
    FROM identities
    WHERE email_verified
    ORDER BY mailbox, linked_at, provider::text
), keepers AS (
    SELECT DISTINCT ON (identities.mailbox) identities.subject
    FROM identities JOIN owners ON owners.mailbox = identities.mailbox AND owners.account_id = identities.account_id
    WHERE identities.provider = 'email'
    ORDER BY identities.mailbox, identities.linked_at, identities.subject
)
UPDATE identities SET subject = 'alias:' || account_id || ':' || subject
WHERE provider = 'email' AND email_verified AND subject NOT IN (SELECT subject FROM keepers);

UPDATE identities SET subject = mailbox
WHERE provider = 'email' AND subject NOT LIKE 'alias:%' AND subject <> mailbox;

DROP INDEX identities_email;
CREATE INDEX identities_mailbox ON identities (mailbox) WHERE email_verified;
