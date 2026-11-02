-- 2026-11-02: the day's conquests were drawn before a ground had to have room for its goal (France was told to take
-- 8,000 tiles in Sudan, which has 3,536). Each one not done yet goes, and its flag draws again at its next take or
-- read, keeping the takes and strikes its day counted. A conquest done keeps its points, and a shield mission, which
-- the bug never touched, stays. Only the day that ran then: a later day was drawn with the room.
DELETE FROM flag_missions AS mission
USING mission_days AS broken
WHERE mission.day = broken.day_id
  AND broken.starts_at <= timestamptz '2026-11-02 06:00:00+00' AND timestamptz '2026-11-02 06:00:00+00' < broken.ends_at
  AND mission.kind = 'take_in' AND NOT mission.done;
