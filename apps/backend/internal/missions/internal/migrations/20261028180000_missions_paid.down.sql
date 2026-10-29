DROP INDEX mission_payments_account;

DELETE FROM outbox WHERE event = 'missions.v1.MissionsPaid';
