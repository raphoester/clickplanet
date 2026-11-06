-- The numbers missions draws with, one document the operator edits: the values the code held until now.
CREATE TABLE rules (
    singleton boolean PRIMARY KEY DEFAULT true CHECK (singleton),
    rules     jsonb   NOT NULL CHECK (jsonb_typeof(rules) = 'object')
);

INSERT INTO rules (rules) VALUES
    ('{"flagMissions": {"goalPercent": 30, "leastGoal": 250, "fewestPoints": 3, "mostPoints": 10}}');
