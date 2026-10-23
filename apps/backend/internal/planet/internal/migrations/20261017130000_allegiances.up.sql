CREATE TABLE allegiances (
    account    uuid        PRIMARY KEY,
    flag       text        NOT NULL CHECK (flag <> ''),
    changed_at timestamptz
);
