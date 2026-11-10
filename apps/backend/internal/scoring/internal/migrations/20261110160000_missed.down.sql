DELETE FROM outbox WHERE event IN ('scoring.v1.RoundMissed', 'scoring.v1.BookMissed');
