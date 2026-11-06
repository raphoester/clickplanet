-- An account with a username: player.v1.PlayerCreated. Every other account is a guest, which no board ranks.
CREATE TABLE players (
    account_id uuid PRIMARY KEY
);

-- An account its player asked to delete: player.v1.PlayerLeaving and PlayerReturned, the latest decision wins.
CREATE TABLE departures (
    account_id uuid        PRIMARY KEY,
    leaving    boolean     NOT NULL,
    decided_at timestamptz NOT NULL
);

-- What player held when scoring stopped asking it, once; player's events keep both from then on.
DO $$
BEGIN
    IF to_regclass('player.profiles') IS NOT NULL THEN
        INSERT INTO players (account_id) SELECT account_id FROM player.profiles ON CONFLICT DO NOTHING;
    END IF;
    IF to_regclass('player.departures') IS NOT NULL THEN
        INSERT INTO departures (account_id, leaving, decided_at)
        SELECT account_id, leaving, decided_at FROM player.departures ON CONFLICT DO NOTHING;
    END IF;
END $$;
