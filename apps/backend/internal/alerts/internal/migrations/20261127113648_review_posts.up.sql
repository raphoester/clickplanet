CREATE TYPE review_cause AS ENUM ('appeal', 'unsure');

-- Each flag the staff were told waits on a person, once by its submission and its cause, kept in the transaction of
-- the post.
CREATE TABLE review_posts (
    submission_id text         NOT NULL,
    cause         review_cause NOT NULL,
    sent_at       timestamptz  NOT NULL,
    PRIMARY KEY (submission_id, cause)
);
