UPDATE messages
SET name = substr(name, length('guest_') + 1)
WHERE sent_at < '2026-09-17 14:31:30+00' AND left(name, 6) = 'guest_';
