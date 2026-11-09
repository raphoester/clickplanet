-- How each battle played, judged once as it starts from its rule set: a siege plays on a blank map.
CREATE TYPE battle_mode AS ENUM ('plain', 'siege');

CREATE TABLE battle_modes (
    round_id text        PRIMARY KEY CHECK (round_id <> ''),
    mode     battle_mode NOT NULL
);

-- Each siege's arena planet told closed: the map from before it is back.
CREATE TABLE closed_arenas (
    round_id text PRIMARY KEY CHECK (round_id <> '')
);
