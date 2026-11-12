ALTER TABLE quiz_answers ADD COLUMN correct boolean, ADD COLUMN timed_out boolean;

UPDATE quiz_answers SET correct = outcome = 'right', timed_out = outcome = 'timed_out';

ALTER TABLE quiz_answers
    ALTER COLUMN correct SET NOT NULL,
    ALTER COLUMN timed_out SET NOT NULL,
    ADD CHECK (NOT (timed_out AND correct)),
    DROP COLUMN outcome;

DROP TYPE quiz_answer_outcome;
