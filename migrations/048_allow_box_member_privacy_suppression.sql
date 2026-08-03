ALTER TABLE privacy_suppressions DROP CONSTRAINT privacy_suppressions_source_check;
ALTER TABLE privacy_suppressions
    ADD CONSTRAINT privacy_suppressions_source_check
        CHECK (source IN ('wellhub', 'totalpass', 'box_member'));
