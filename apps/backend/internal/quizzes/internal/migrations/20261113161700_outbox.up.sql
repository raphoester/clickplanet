-- QuizAnswered waits here with the answer until Redis takes it: a right answer told is a right answer paid.
CREATE TYPE outbox_event AS ENUM ('quizzes.v1.QuizAnswered');

CREATE TABLE outbox (
    position bigserial    PRIMARY KEY,
    event    outbox_event NOT NULL,
    payload  bytea        NOT NULL
);
