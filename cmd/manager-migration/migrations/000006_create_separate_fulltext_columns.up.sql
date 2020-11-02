ALTER TABLE transaction_events ADD COLUMN fulltext_col tsvector;
CREATE INDEX transaction_txts_idx ON transaction_events USING GIN (fulltext_col);
