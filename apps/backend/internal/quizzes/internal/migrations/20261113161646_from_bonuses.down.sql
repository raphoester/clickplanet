-- The answers and the ratings go back to the bonuses module where its schema still is.
DO $$
DECLARE
    moved text;
BEGIN
    IF to_regnamespace('bonuses') IS NOT NULL THEN
        FOREACH moved IN ARRAY ARRAY['quiz_answers', 'quiz_questions'] LOOP
            EXECUTE format('ALTER TABLE %I SET SCHEMA bonuses', moved);
        END LOOP;
        ALTER TYPE quiz_answer_outcome SET SCHEMA bonuses;
    ELSE
        DROP TABLE quiz_questions;
        DROP TABLE quiz_answers;
        DROP TYPE quiz_answer_outcome;
    END IF;
END $$;

DROP TABLE rule_set_choice;
DROP TABLE rule_sets;
