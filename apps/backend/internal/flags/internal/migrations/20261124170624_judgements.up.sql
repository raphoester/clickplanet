CREATE TYPE judgement_verdict AS ENUM ('approve', 'reject', 'unsure');

-- What the judge, a Claude model, ruled on a submission as it waited: an approval or a rejection it applied, or unsure,
-- which left it to the operator. Kept for the operator's audit; a submission replaced takes its judgement with it.
CREATE TABLE judgements (
    submission_id uuid              PRIMARY KEY REFERENCES submissions (id) ON DELETE CASCADE,
    verdict       judgement_verdict NOT NULL,
    why           text              NOT NULL,
    judged_at     timestamptz       NOT NULL
);
