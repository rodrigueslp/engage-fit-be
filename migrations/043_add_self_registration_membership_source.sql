ALTER TABLE students
    DROP CONSTRAINT students_membership_started_source_check;

ALTER TABLE students
    ADD CONSTRAINT students_membership_started_source_check CHECK (
        membership_started_source IS NULL OR membership_started_source IN (
            'manual',
            'integration',
            'first_checkin_inferred',
            'self_registration'
        )
    );
