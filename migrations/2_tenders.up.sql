BEGIN;

DO $$
BEGIN
    IF NOT EXISTS (
        SELECT 1 
        FROM pg_type 
        WHERE typname = 'service_type'
    ) THEN
        CREATE TYPE service_type AS ENUM (
            'Construction',
            'Delivery',
            'Manufacture'
        );
    END IF;
END $$;

DO $$
BEGIN
    IF NOT EXISTS (
        SELECT 1 
        FROM pg_type 
        WHERE typname = 'tender_status_type'
    ) THEN
        CREATE TYPE tender_status_type AS ENUM (
            'Created',
            'Published',
            'Closed'
        );
    END IF;
END $$;

CREATE TABLE IF NOT EXISTS tender (
    id UUID PRIMARY KEY DEFAULT uuid_generate_v4(),
    organization_id UUID REFERENCES organization(id) ON DELETE CASCADE,
    name VARCHAR(100) NOT NULL,
    description VARCHAR(500) NOT NULL,
    type service_type,
    status tender_status_type,
    version integer,
    created_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP
);

COMMIT;