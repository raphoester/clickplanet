CREATE TABLE rule_baselines (
    seq        bigint      GENERATED ALWAYS AS IDENTITY,
    request    text        PRIMARY KEY CHECK (request <> ''),
    rule_set   text        NOT NULL REFERENCES rule_sets (name),
    applied_at timestamptz NOT NULL
);

CREATE INDEX rule_baselines_latest ON rule_baselines (applied_at DESC, seq DESC);

CREATE TABLE rule_override (
    only_one  boolean     PRIMARY KEY DEFAULT true CHECK (only_one),
    rule_set  text        NOT NULL REFERENCES rule_sets (name),
    starts_at timestamptz NOT NULL,
    ends_at   timestamptz NOT NULL CHECK (ends_at >= starts_at)
);
