-- Brittany left the build in the deploy that sells it: heard open at once, so its players never see it unknown while
-- the flags module's restatement at boot travels.
INSERT INTO heard_flags (flag, state, decided_at) VALUES ('xb', 'open', now()) ON CONFLICT (flag) DO NOTHING;
