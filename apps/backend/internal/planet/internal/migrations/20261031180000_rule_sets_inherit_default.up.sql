-- Default holds every value; it was written before the map could freeze.
UPDATE rule_sets SET rules = '{"frozen": false}' || rules WHERE name = 'default';

-- What `kept` changes from `defaults`, read as a set was read before: a value left out, or null, was zero.
CREATE FUNCTION pg_temp.own_values(kept jsonb, defaults jsonb) RETURNS jsonb LANGUAGE plpgsql AS $$
DECLARE
    own   jsonb := '{}';
    field text;
    base  jsonb;
    held  jsonb;
    below jsonb;
BEGIN
    FOR field, base IN SELECT key, value FROM jsonb_each(defaults) LOOP
        held := coalesce(kept -> field, 'null');
        IF jsonb_typeof(base) = 'object' THEN
            below := pg_temp.own_values(CASE WHEN held = 'null' THEN '{}' ELSE held END, base);
            IF below <> '{}' THEN
                own := own || jsonb_build_object(field, below);
            END IF;
            CONTINUE;
        END IF;

        IF held = 'null' THEN
            held := CASE jsonb_typeof(base)
                WHEN 'number' THEN '0'::jsonb
                WHEN 'string' THEN '"0s"'::jsonb
                WHEN 'boolean' THEN 'false'::jsonb
                WHEN 'array' THEN '[]'::jsonb
            END;
        END IF;
        IF held IS DISTINCT FROM base THEN
            own := own || jsonb_build_object(field, held);
        END IF;
    END LOOP;

    RETURN own || coalesce((SELECT jsonb_object_agg(key, value) FROM jsonb_each(kept)
                            WHERE NOT defaults ? key AND value <> 'null'), '{}');
END
$$;

-- Each other set holds only what it changes: a value it held equal to default's now follows default.
UPDATE rule_sets SET rules = pg_temp.own_values(rules, (SELECT rules FROM rule_sets WHERE name = 'default'))
WHERE name <> 'default';

DROP FUNCTION pg_temp.own_values(jsonb, jsonb);
