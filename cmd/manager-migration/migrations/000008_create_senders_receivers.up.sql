CREATE INDEX transaction_senders_idx ON transaction_events USING GIN (senders);
CREATE INDEX transaction_recipients_idx ON transaction_events USING GIN (recipients);
