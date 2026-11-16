-- The missions each account has done, by kind, from missions.v1.MissionsDone: {"take_for": {"done": 3, "goals": 540}}.
-- The XP of a level is paid on the goals. The missions done before are told again by the operator's
-- missions.v1.AdminService/RetellMissionsDone.
ALTER TABLE stats ADD COLUMN IF NOT EXISTS missions_by_kind jsonb NOT NULL DEFAULT '{}';
