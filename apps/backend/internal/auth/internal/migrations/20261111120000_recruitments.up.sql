CREATE TABLE recruitments (
    arrival_id   uuid PRIMARY KEY,
    recruiter_id uuid
);

CREATE INDEX recruitments_recruiter_id ON recruitments (recruiter_id);
