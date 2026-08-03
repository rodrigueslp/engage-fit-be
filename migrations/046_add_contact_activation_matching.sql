ALTER TABLE contact_activation_requests
    ADD COLUMN match_strategy VARCHAR(40) NOT NULL DEFAULT '';

ALTER TABLE contact_activation_requests
    DROP CONSTRAINT contact_activation_requests_status_check;

ALTER TABLE contact_activation_requests
    ADD CONSTRAINT contact_activation_requests_status_check
        CHECK (status IN ('awaiting_message', 'confirmed', 'pending_sync', 'needs_review', 'expired', 'cancelled'));
