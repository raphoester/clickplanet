-- A set is a quizzes.v1.Rules as protojson: rules.Snapshot.Restore is the check. "default" holds every value, another
-- set what it changes.
CREATE TABLE rule_sets (
    name  text  PRIMARY KEY CHECK (name <> ''),
    rules jsonb NOT NULL CHECK (jsonb_typeof(rules) = 'object')
);

-- The rule set the schedule told the quizzes to play last, which they play at boot. One row at most.
CREATE TABLE rule_set_choice (
    single     boolean     PRIMARY KEY DEFAULT true CHECK (single),
    rule_set   text        NOT NULL CHECK (rule_set <> ''),
    decided_at timestamptz NOT NULL
);

-- The bonuses module kept the quizzes' half of each of its sets under "quiz", beside "activeWithin", "welcome" and
-- "frozen", which a quiz read too, until the quizzes were a module of their own: each set comes across under its name.
-- The bonuses module's migrations of the same release leave its sets as they are, so this reads them whichever module
-- migrates first. A value a set held as null, which read as unset, is left out.
DO $$
BEGIN
    IF to_regclass('bonuses.rule_sets') IS NOT NULL THEN
        INSERT INTO rule_sets (name, rules)
        SELECT name, jsonb_strip_nulls(jsonb_build_object(
            'enabled', q -> 'enabled',
            'minInterval', q -> 'minInterval',
            'maxInterval', q -> 'maxInterval',
            'missRetry', q -> 'missRetry',
            'offerTtl', q -> 'offerTtl',
            'answerWindow', q -> 'answerWindow',
            'leaderBias', q -> 'leaderBias',
            'maxRightAnswersPerHour', q -> 'maxChargesPerHour',
            'target', q -> 'target',
            'width', q -> 'width',
            'floor', q -> 'floor',
            'floorAfter', q -> 'floorAfter',
            'explore', q -> 'explore',
            'activeWithin', played -> 'activeWithin',
            'welcome', CASE WHEN played -> 'welcome' ? 'within'
                THEN jsonb_build_object('within', played -> 'welcome' -> 'within') END,
            'frozen', played -> 'frozen'
        ))
        FROM (
            SELECT name, rules AS played, COALESCE(rules -> 'quiz', '{}'::jsonb) AS q FROM bonuses.rule_sets
        ) kept;
    END IF;
END $$;

-- A database with no bonuses rules to copy: production's quiz half of 13 November 2026.
INSERT INTO rule_sets (name, rules) VALUES ('default', '{
    "enabled": true, "minInterval": "240s", "maxInterval": "480s", "missRetry": "120s", "offerTtl": "25s",
    "answerWindow": "8s", "leaderBias": 1, "maxRightAnswersPerHour": 12,
    "target": 0.7, "width": 1, "floor": 0.42, "floorAfter": 15, "explore": 0.1,
    "activeWithin": "120s",
    "welcome": {"within": "600s"},
    "frozen": false
}') ON CONFLICT (name) DO NOTHING;

-- The answers counted and each question's rating move from the bonuses module, rows, indexes and constraints with them,
-- so nothing is copied and nothing is lost. A fresh database moves the ones the bonuses module's migrations just made.
DO $$
DECLARE
    moved text;
BEGIN
    IF to_regtype('bonuses.quiz_answer_outcome') IS NOT NULL THEN
        EXECUTE format('ALTER TYPE bonuses.quiz_answer_outcome SET SCHEMA %I', current_schema());
    END IF;
    FOREACH moved IN ARRAY ARRAY['quiz_answers', 'quiz_questions'] LOOP
        IF to_regclass('bonuses.' || moved) IS NOT NULL THEN
            EXECUTE format('ALTER TABLE bonuses.%I SET SCHEMA %I', moved, current_schema());
        END IF;
    END LOOP;
END $$;

DO $$
BEGIN
    CREATE TYPE quiz_answer_outcome AS ENUM ('right', 'wrong', 'timed_out');
EXCEPTION WHEN duplicate_object THEN NULL;
END $$;

-- Every answer the quizzes' difficulty counts, one row each, never edited: an opened question answered right or wrong,
-- a timeout the page sent, or an answer past the deadline. It is the source of truth: the ratings are recomputed from
-- it with a better model. A caller with no account answers with none, and a forgotten account's id is blanked.
CREATE TABLE IF NOT EXISTS quiz_answers (
    answer_id   bigint              GENERATED ALWAYS AS IDENTITY PRIMARY KEY,
    question_id text                NOT NULL CHECK (question_id <> ''),
    account_id  uuid,
    -- From the opening, which stamped the deadline.
    answer_ms   integer             NOT NULL CHECK (answer_ms >= 0),
    answered_at timestamptz         NOT NULL,
    outcome     quiz_answer_outcome NOT NULL
);

CREATE INDEX IF NOT EXISTS quiz_answers_account ON quiz_answers (account_id) WHERE account_id IS NOT NULL;

-- One row per question answered at least once: the log's running totals, and the rating its answers moved it to from
-- its prior. A question with no row is at its prior, the bank's.
CREATE TABLE IF NOT EXISTS quiz_questions (
    question_id   text             PRIMARY KEY CHECK (question_id <> ''),
    rating        double precision NOT NULL CHECK (rating BETWEEN -10 AND 10),
    answers       integer          NOT NULL CHECK (answers > 0),
    right_answers integer          NOT NULL CHECK (right_answers >= 0 AND right_answers <= answers)
);
