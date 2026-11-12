-- Postgres cannot drop an enum value: the rounds that used it go back to the gain they were scored on.
UPDATE rounds SET ranking = 'gain' WHERE ranking = 'fighters';
