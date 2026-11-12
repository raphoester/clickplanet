-- Every answer the quizzes' difficulty counts, one row each, never edited: an opened question answered right or wrong,
-- a timeout the page sent, or an answer past the deadline. A banner never opened and a question opened then left to
-- lapse are not answers. It is the source of truth: the ratings are recomputed from it with a better model. A caller with
-- no account answers with none, and a forgotten account's id is blanked (player.v1.PlayerForgotten).
CREATE TABLE quiz_answers (
    answer_id   bigint      GENERATED ALWAYS AS IDENTITY PRIMARY KEY,
    question_id text        NOT NULL CHECK (question_id <> ''),
    account_id  uuid,
    correct     boolean     NOT NULL,
    timed_out   boolean     NOT NULL CHECK (NOT (timed_out AND correct)),
    -- From the opening, which stamped the deadline.
    answer_ms   integer     NOT NULL CHECK (answer_ms >= 0),
    answered_at timestamptz NOT NULL
);

CREATE INDEX quiz_answers_account ON quiz_answers (account_id) WHERE account_id IS NOT NULL;

-- One row per question answered at least once: the log's running totals, and the rating its answers moved it to from
-- its prior. A question with no row is at its prior, the bank's.
CREATE TABLE quiz_questions (
    question_id   text             PRIMARY KEY CHECK (question_id <> ''),
    rating        double precision NOT NULL CHECK (rating BETWEEN -10 AND 10),
    answers       integer          NOT NULL CHECK (answers > 0),
    right_answers integer          NOT NULL CHECK (right_answers >= 0 AND right_answers <= answers)
);
