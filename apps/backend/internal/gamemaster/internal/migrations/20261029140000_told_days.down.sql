DELETE FROM outbox WHERE event = 'gamemaster.v1.DayStarted';

DROP TABLE told_days;
