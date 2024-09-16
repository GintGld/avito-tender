BEGIN;

CREATE TABLE IF NOT EXISTS rollback_tender(
    id UUID REFERENCES tender(id) ON DELETE CASCADE,
    organization_id UUID REFERENCES organization(id) ON DELETE CASCADE,
    name VARCHAR(100) NOT NULL,
    description VARCHAR(500) NOT NULL,
    type service_type,
    status tender_status_type,
    version integer,
    created_at TIMESTAMP,
    PRIMARY KEY(id, version)
);

CREATE TABLE IF NOT EXISTS rollback_bid(
    id UUID REFERENCES bid(id) ON DELETE CASCADE,
    tender_id UUID REFERENCES tender(id) ON DELETE CASCADE,
    name VARCHAR(100) NOT NULL,
    description VARCHAR(500) NOT NULL,
    status bid_status_type,
    author_type author_type,
    author_id UUID,
    version integer,
    created_at TIMESTAMP,
    PRIMARY KEY(id, version)
);

COMMIT;