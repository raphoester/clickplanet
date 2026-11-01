-- What `kept` changes from `defaults`, read as a set was read before: a value left out, or null, was zero. A bound unset
-- never read, and no set can say so over a default that holds it.
CREATE FUNCTION pg_temp.own_values(name text, kept jsonb, defaults jsonb) RETURNS jsonb LANGUAGE plpgsql AS $$
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
            below := pg_temp.own_values(name, CASE WHEN held = 'null' THEN '{}' ELSE held END, base);
            IF below <> '{}' THEN
                own := own || jsonb_build_object(field, below);
            END IF;
            CONTINUE;
        END IF;

        IF held = 'null' AND field IN ('maxSkew', 'certainSkew', 'minCoherence', 'certainCoherence') THEN
            RAISE EXCEPTION 'the rule set % leaves % unset, which default holds: set it in default too, or unset it there', name, field;
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
UPDATE rule_sets SET rules = pg_temp.own_values(name, rules, (SELECT rules FROM rule_sets WHERE name = 'default'))
WHERE name <> 'default';

DROP FUNCTION pg_temp.own_values(text, jsonb, jsonb);
