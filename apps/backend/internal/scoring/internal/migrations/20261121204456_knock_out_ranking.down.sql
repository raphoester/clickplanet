-- Postgres cannot drop an enum value: the rounds that used it go back to the fighters' ranking.
UPDATE rounds SET ranking = 'fighters' WHERE ranking = 'knock_out';
