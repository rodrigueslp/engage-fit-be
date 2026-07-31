ALTER TABLE contact_activation_requests
    ADD COLUMN is_new_student BOOLEAN NOT NULL DEFAULT FALSE;
