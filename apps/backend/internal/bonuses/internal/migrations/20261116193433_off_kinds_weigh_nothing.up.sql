-- The draw ignores the hand since planet answers no room: a kind planet's set turns off would be drawn, and its wallet
-- would keep none of it. Each set whose planet set of the same name turns bombs or enclosures off weighs them 0, so its
-- boxes and right answers are drawn among the kinds that play (capital-siege's bombs; its enclosures weigh 0 already).
DO $$
BEGIN
    IF to_regclass('planet.rule_sets') IS NOT NULL THEN
        UPDATE rule_sets AS b
        SET rules = jsonb_set(b.rules, '{kinds}', coalesce(b.rules -> 'kinds', '{}') || '{"bomb": 0}')
        FROM planet.rule_sets AS p
        WHERE p.name = b.name AND coalesce((p.rules -> 'charges' -> 'bomb' ->> 'off')::boolean, false);

        UPDATE rule_sets AS b
        SET rules = jsonb_set(b.rules, '{kinds}', coalesce(b.rules -> 'kinds', '{}') || '{"encloseClicks": 0}')
        FROM planet.rule_sets AS p
        WHERE p.name = b.name AND coalesce((p.rules -> 'charges' -> 'enclose' ->> 'off')::boolean, false);
    END IF;
END $$;
