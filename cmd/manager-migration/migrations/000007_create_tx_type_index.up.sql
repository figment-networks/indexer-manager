CREATE INDEX transaction_types_idx ON transaction_events USING GIN (type);
