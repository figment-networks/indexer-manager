ALTER TABLE transaction_events DROP COLUMN fee;
ALTER TABLE transaction_events ADD COLUMN fee JSONB;
