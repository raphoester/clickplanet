DELETE FROM outbox WHERE event IN ('player.v1.PlayerLeaving', 'player.v1.PlayerReturned', 'player.v1.PlayerForgotten');
