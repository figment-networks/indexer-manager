CREATE EXTENSION IF NOT EXISTS "uuid-ossp";

CREATE TABLE IF NOT EXISTS transaction_events
(
    id         uuid DEFAULT uuid_generate_v4 (),
    created_at TIMESTAMP WITH TIME ZONE DEFAULT CURRENT_TIMESTAMP,
    updated_at TIMESTAMP WITH TIME ZONE,

    network         VARCHAR(100)  NOT NULL,
    chain_id        VARCHAR(100)  NOT NULL,

    height          DECIMAL(65, 0) NOT NULL,
    hash            TEXT    NOT NULL,
    block_hash      TEXT    NOT NULL,
    time            TIMESTAMP WITH TIME ZONE,

    type        VARCHAR(100) NOT NULL,
    senders     TEXT[],
    recipients  TEXT[],
    amount      DECIMAL(65, 0),
    fee         DECIMAL(65, 0),

    gas_wanted  DECIMAL(65, 0)           NOT NULL,
    gas_used  DECIMAL(65, 0)           NOT NULL,

    data    JSONB,
    memo    TEXT,

    PRIMARY KEY (id)
);

CREATE index idx_tx_ev_height on transaction_events (network, chain_id, height);
CREATE index idx_tx_ev_hash on transaction_events (network, chain_id, hash);
CREATE index idx_tx_ev_block_hash on transaction_events (network, chain_id, block_hash);


