CREATE TABLE knockouts (
    map            text        NOT NULL CHECK (map <> 'world'),
    flag           text        NOT NULL,
    by_flag        text        NOT NULL,
    knocked_out_at timestamptz NOT NULL,
    PRIMARY KEY (map, flag)
);
