-- The chat members a person silenced. Keyed on the author tag, which is what the
-- chat shows and what every message already carries, so a ban needs no address.
--
-- Nothing here deletes a message: the text stays in messages, and the ban is
-- what blanks it on the way out.
CREATE TABLE bans (
    tag       text        PRIMARY KEY,
    banned_at timestamptz NOT NULL,
    reason    text        NOT NULL
);
