CREATE TABLE IF NOT EXISTS schedule_latest
(
    id          uuid DEFAULT uuid_generate_v4(),
    time        TIMESTAMP WITH TIME ZONE,

    network     VARCHAR(100)  NOT NULL,
    chain_id    VARCHAR(100)  NOT NULL,
    version     VARCHAR(50)  NOT NULL,
    kind        VARCHAR(100),


    hash        TEXT,
    height      BIGSERIAL,
    epoch       TEXT,
    nonce       BYTEA,

    PRIMARY KEY (id)
);


CREATE INDEX idx_sch_lst_nvc on schedule_latest(network, chain_id, version,  kind, time);
