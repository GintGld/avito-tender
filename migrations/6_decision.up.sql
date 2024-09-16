BEGIN;

DO $$
BEGIN
    IF NOT EXISTS (
        SELECT 1 
        FROM pg_type 
        WHERE typname = 'tender_status_type'
    ) THEN
        CREATE TYPE decision_type AS ENUM (
            'Approved',
            'Declined'
        );
    END IF;
END $$;

CREATE TABLE IF NOT EXISTS decision(
    user_id UUID REFERENCES employee(id) ON DELETE CASCADE,
    bid_id UUID REFERENCES bid(id) ON DELETE CASCADE,
    decision decision_type,
    PRIMARY KEY(user_id,bid)
);

COMMIT;