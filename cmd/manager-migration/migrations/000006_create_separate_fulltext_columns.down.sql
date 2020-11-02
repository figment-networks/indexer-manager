DROP INDEX IF EXISTS transaction_txts_idx;
ALTER TABLE transaction_events DROP COLUMN fulltext_col;
