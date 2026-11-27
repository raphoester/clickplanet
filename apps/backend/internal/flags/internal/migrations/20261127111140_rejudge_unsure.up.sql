-- The review rule grew on 2026-11-27: a historical, fictional or invented flag opens too. A flag the judge left unsure
-- under the old rule and that still waits is judged again under the new one.
DELETE FROM judgements j USING submissions s
WHERE j.submission_id = s.id AND j.verdict = 'unsure' AND s.state = 'waiting';
