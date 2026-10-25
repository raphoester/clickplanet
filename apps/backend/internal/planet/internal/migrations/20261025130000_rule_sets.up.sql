CREATE TABLE rule_sets (
    name                     text             PRIMARY KEY CHECK (name <> ''),
    tempo                    double precision NOT NULL CHECK (tempo > 0),
    box_min_interval_ms      bigint           NOT NULL,
    box_max_interval_ms      bigint           NOT NULL,
    box_max_charges_per_hour integer          NOT NULL,
    crowd                    integer          NOT NULL
);

-- The game's own rules on 25 October 2026: a refill at rateLimiter's rate, a box every 4 to 8 minutes, 12 charges an
-- hour, and a crowd of 6 networks.
INSERT INTO rule_sets (name, tempo, box_min_interval_ms, box_max_interval_ms, box_max_charges_per_hour, crowd)
VALUES ('default', 1, 240000, 480000, 12, 6);
