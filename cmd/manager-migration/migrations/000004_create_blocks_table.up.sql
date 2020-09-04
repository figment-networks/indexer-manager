CREATE EXTENSION IF NOT EXISTS "uuid-ossp";

CREATE TABLE IF NOT EXISTS blocks
(
    id         uuid DEFAULT uuid_generate_v4(),
    time       TIMESTAMP WITH TIME ZONE NOT NULL,

    created_at TIMESTAMP WITH TIME ZONE DEFAULT CURRENT_TIMESTAMP,
    updated_at TIMESTAMP WITH TIME ZONE,

    network         VARCHAR(100)  NOT NULL,
    chain_id        VARCHAR(100)  NOT NULL,
    version         VARCHAR(100)  NOT NULL,

    height          DECIMAL(65, 0) NOT NULL,
    epoch           TEXT    NOT NULL,
    hash            TEXT    NOT NULL,

    numtxs          DECIMAL(65, 0) NOT NULL DEFAULT 0,

    PRIMARY KEY (id)
);


CREATE INDEX idx_blx_height on blocks (network, chain_id, epoch, height);
CREATE INDEX idx_blx_time on blocks (network, chain_id, time);
CREATE UNIQUE INDEX idx_blx_hash on blocks (network, chain_id, epoch, hash);
