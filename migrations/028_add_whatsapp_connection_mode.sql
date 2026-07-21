DO $$
BEGIN
    IF NOT EXISTS (
        SELECT 1
        FROM information_schema.columns
        WHERE table_schema = 'public'
          AND table_name = 'whatsapp_settings'
          AND column_name = 'connection_mode'
    ) THEN
        ALTER TABLE whatsapp_settings
            ADD COLUMN connection_mode VARCHAR(20) NOT NULL DEFAULT 'platform';

        -- Existing configured connections keep their current behavior after the migration.
        -- New academies have no row yet and therefore use the EngageFit platform connection.
        UPDATE whatsapp_settings
        SET connection_mode = 'dedicated'
        WHERE enabled = TRUE
          AND TRIM(instance_name) <> ''
          AND TRIM(api_key_encrypted) <> '';
    END IF;
END $$;

ALTER TABLE whatsapp_settings
    DROP CONSTRAINT IF EXISTS whatsapp_settings_connection_mode_check;

ALTER TABLE whatsapp_settings
    ADD CONSTRAINT whatsapp_settings_connection_mode_check
    CHECK (connection_mode IN ('platform', 'dedicated'));
