DELETE FROM outbox WHERE event = 'auth.v1.DiscordJoinGranted';

DROP TABLE discord_grants;
