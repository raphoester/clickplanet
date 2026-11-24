-- England, Scotland and Wales left the build in the deploy that sells them: heard open at once, on their pins, so their
-- players never see them unknown, nor an arena opened while the flags module's restatement at boot travels seat them
-- nowhere.
INSERT INTO heard_flags (flag, state, decided_at, pin_latitude, pin_longitude, pinned_at)
VALUES
    ('gb-eng', 'open', now(), 51.501941, -0.118668, now()),
    ('gb-sct', 'open', now(), 55.948328, -3.219091, now()),
    ('gb-wls', 'open', now(), 51.499995, -3.225008, now())
ON CONFLICT (flag) DO NOTHING;
