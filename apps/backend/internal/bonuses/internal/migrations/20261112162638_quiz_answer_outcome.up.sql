CREATE TYPE quiz_answer_outcome AS ENUM ('right', 'wrong', 'timed_out');

ALTER TABLE quiz_answers ADD COLUMN outcome quiz_answer_outcome;

UPDATE quiz_answers SET outcome = CASE
    WHEN correct THEN 'right'
    WHEN timed_out THEN 'timed_out'
    ELSE 'wrong'
END::quiz_answer_outcome;

ALTER TABLE quiz_answers
    ALTER COLUMN outcome SET NOT NULL,
    DROP COLUMN correct,
    DROP COLUMN timed_out;
