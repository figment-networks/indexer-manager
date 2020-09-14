CREATE EXTENSION IF NOT EXISTS "uuid-ossp";

CREATE TABLE IF NOT EXISTS schedule
(
    id          uuid DEFAULT uuid_generate_v4(),

    runID       uuid NOT NULL,

    network     VARCHAR(100)  NOT NULL,
    chain_id    VARCHAR(100)  NOT NULL,
    version     VARCHAR(50)  NOT NULL,

    duration    BIGINT,
    kind        TEXT,

    PRIMARY KEY (id)
);

CREATE INDEX idx_sch_runID on schedule(runID);
