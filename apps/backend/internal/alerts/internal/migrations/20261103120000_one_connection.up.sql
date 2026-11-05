-- A player connects Discord by signing in with it: the alerts module starts no Discord trip of its own.
DROP TABLE link_flows;

-- 0 is a role never painted, which the boot paints in its flag's color.
ALTER TABLE roles ADD COLUMN color integer NOT NULL DEFAULT 0;
