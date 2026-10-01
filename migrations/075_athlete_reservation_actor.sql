ALTER TABLE reservation_audit_events
    ALTER COLUMN actor_user_id DROP NOT NULL,
    ADD COLUMN actor_athlete_account_id UUID REFERENCES athlete_accounts(id) ON DELETE RESTRICT;

ALTER TABLE reservation_audit_events
    ADD CONSTRAINT reservation_audit_actor_check CHECK (
        (actor_user_id IS NOT NULL AND actor_athlete_account_id IS NULL) OR
        (actor_user_id IS NULL AND actor_athlete_account_id IS NOT NULL)
    );
