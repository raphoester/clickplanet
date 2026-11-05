-- Each set holds every value again: default's under its own.
CREATE FUNCTION pg_temp.played(defaults jsonb, own jsonb) RETURNS jsonb LANGUAGE plpgsql AS $$
DECLARE
    played jsonb := defaults;
    field  text;
    held   jsonb;
BEGIN
    FOR field, held IN SELECT key, value FROM jsonb_each(own) LOOP
        IF jsonb_typeof(held) = 'object' AND jsonb_typeof(defaults -> field) = 'object' THEN
            held := pg_temp.played(defaults -> field, held);
        END IF;
        played := played || jsonb_build_object(field, held);
    END LOOP;
    RETURN played;
END
$$;

UPDATE rule_sets SET rules = pg_temp.played((SELECT rules FROM rule_sets WHERE name = 'default'), rules)
WHERE name <> 'default';

DROP FUNCTION pg_temp.played(jsonb, jsonb);
