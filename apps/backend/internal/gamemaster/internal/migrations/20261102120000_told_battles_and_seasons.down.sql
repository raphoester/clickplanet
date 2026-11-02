DELETE FROM outbox WHERE event IN ('gamemaster.v1.BattleStarted', 'gamemaster.v1.SeasonStarted');

DROP TABLE told_seasons;
DROP TABLE told_battles;
