-- Every name in the log gets the "guest_" prefix, as the chat has sent it since usernames came.
--
-- Usernames went live with the deploy that finished at 2026-09-17 14:31:45 UTC, and the binary before it knew
-- no usernames, so every message sent before it came from a guest. The last one was sent at 14:11, so nothing
-- sent under a username is before the cutoff.
UPDATE messages
SET name = 'guest_' || name
WHERE sent_at < '2026-09-17 14:31:30+00';
