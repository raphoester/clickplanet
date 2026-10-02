UPDATE messages
SET name = 'guest_' || name
WHERE sent_at < '2026-09-17 14:31:30+00';
