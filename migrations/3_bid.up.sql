BEGIN;

DO $$
BEGIN
    IF NOT EXISTS (
        SELECT 1 
        FROM pg_type 
        WHERE typname = 'bid_status_type'
    ) THEN
        CREATE TYPE bid_status_type AS ENUM(
            'Created',
            'Published',
            'Canceled'
        );
    END IF;
END $$;

DO $$
BEGIN
    IF NOT EXISTS (
        SELECT 1 
        FROM pg_type 
        WHERE typname = 'author_type'
    ) THEN
        CREATE TYPE author_type AS ENUM(
            'User',
            'Organization'
        );
    END IF;
END $$;

CREATE TABLE IF NOT EXISTS bid(
    id UUID PRIMARY KEY DEFAULT uuid_generate_v4(),
    tender_id UUID REFERENCES tender(id) ON DELETE CASCADE,
    name VARCHAR(100) NOT NULL,
    description VARCHAR(500) NOT NULL,
    status bid_status_type,
    author_type author_type,
    author_id UUID,
    version integer,
    created_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP
);

COMMIT;