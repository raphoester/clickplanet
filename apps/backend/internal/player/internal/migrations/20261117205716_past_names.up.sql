-- Every username a player chose to leave for another, from 2026-11-17: nothing kept the earlier ones. The public card
-- shows the last few (GetPlayer). Forgotten with the account.
CREATE TABLE past_names (
    id          bigserial   PRIMARY KEY,
    account_id  uuid        NOT NULL,
    name        text        NOT NULL,
    -- The game's fold (NFKC_Casefold, players.Name.Folded): a name is shown once, and never the one held now.
    name_folded text        NOT NULL,
    left_at     timestamptz NOT NULL
);

CREATE INDEX past_names_account_idx ON past_names (account_id, left_at DESC);
