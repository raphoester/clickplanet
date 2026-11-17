-- Missions records no planet.v1.GrantCharges since it pays XP: the relay sent each one left, and none can stop the cast.
DELETE FROM outbox WHERE event = 'planet.v1.GrantCharges';
ALTER TYPE outbox_event RENAME TO outbox_event_before;
CREATE TYPE outbox_event AS ENUM ('missions.v1.MissionsPaid', 'scoring.v1.GrantPoints', 'missions.v1.FlagMissionDone',
    'missions.v1.MissionsDone');
ALTER TABLE outbox ALTER COLUMN event TYPE outbox_event USING event::text::outbox_event;
DROP TYPE outbox_event_before;
