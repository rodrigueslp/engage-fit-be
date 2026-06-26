CREATE OR REPLACE FUNCTION set_updated_at()
RETURNS TRIGGER AS $$
BEGIN
    NEW.updated_at = NOW();
    RETURN NEW;
END;
$$ LANGUAGE plpgsql;

CREATE TRIGGER set_boxes_updated_at
BEFORE UPDATE ON boxes
FOR EACH ROW
EXECUTE FUNCTION set_updated_at();

CREATE TRIGGER set_users_updated_at
BEFORE UPDATE ON users
FOR EACH ROW
EXECUTE FUNCTION set_updated_at();

CREATE TRIGGER set_students_updated_at
BEFORE UPDATE ON students
FOR EACH ROW
EXECUTE FUNCTION set_updated_at();

CREATE TRIGGER set_campaigns_updated_at
BEFORE UPDATE ON campaigns
FOR EACH ROW
EXECUTE FUNCTION set_updated_at();

CREATE TRIGGER set_whatsapp_settings_updated_at
BEFORE UPDATE ON whatsapp_settings
FOR EACH ROW
EXECUTE FUNCTION set_updated_at();

CREATE TRIGGER set_message_templates_updated_at
BEFORE UPDATE ON message_templates
FOR EACH ROW
EXECUTE FUNCTION set_updated_at();
