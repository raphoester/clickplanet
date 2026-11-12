CREATE TABLE IF NOT EXISTS recruitments (
    arrival_id   uuid PRIMARY KEY,
    recruiter_id uuid
);

CREATE INDEX IF NOT EXISTS recruitments_recruiter_id ON recruitments (recruiter_id);
