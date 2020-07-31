CREATE TABLE IF NOT EXISTS transactions
(
    id         BIGSERIAL                NOT NULL,
    created_at TIMESTAMP WITH TIME ZONE NOT NULL,
    updated_at TIMESTAMP WITH TIME ZONE NOT NULL,

    height       DECIMAL(65, 0)                     NOT NULL,
    hash       TEXT                     NOT NULL,
    gas_wanted  DECIMAL(65, 0)           NOT NULL,
    gas_used  DECIMAL(65, 0)           NOT NULL,
    memo  TEXT,


    PRIMARY KEY (id)
);

CREATE index idx_tx_height on transactions (height);
CREATE index idx_tx_hash on transactions (hash);