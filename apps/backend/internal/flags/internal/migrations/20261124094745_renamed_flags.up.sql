-- Two flags never share a name, whatever its case.
CREATE UNIQUE INDEX flags_name_key ON flags (lower(name));

ALTER TYPE outbox_event ADD VALUE 'flags.v1.FlagRenamed';
