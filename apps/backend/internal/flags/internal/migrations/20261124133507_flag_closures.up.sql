-- Why the operator closed a flag before its end, in its own words, which the flag's founder reads. A flag closed at its
-- end has none, and one that comes back loses it. The flags closed before this ran keep none.
CREATE TABLE closures (
    flag text PRIMARY KEY REFERENCES flags (id) ON DELETE CASCADE,
    note text NOT NULL
);
