-- A payment the rules refused, which the shop refunds.
ALTER TYPE outbox_event ADD VALUE 'flags.v1.TimeRefused';
