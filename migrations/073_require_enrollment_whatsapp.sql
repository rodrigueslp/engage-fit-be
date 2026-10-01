CREATE OR REPLACE FUNCTION require_enrollment_person_phone()
RETURNS TRIGGER AS $$
DECLARE
    enrollment_phone VARCHAR(16);
BEGIN
    SELECT phone_e164
      INTO enrollment_phone
      FROM persons
     WHERE id = NEW.person_id
       AND organization_id = NEW.organization_id
     FOR SHARE;

    IF enrollment_phone IS NULL OR btrim(enrollment_phone) = '' THEN
        RAISE EXCEPTION USING
            ERRCODE = '23514',
            CONSTRAINT = 'enrollments_person_phone_required',
            MESSAGE = 'an enrollment requires a person WhatsApp phone';
    END IF;

    RETURN NEW;
END;
$$ LANGUAGE plpgsql;

CREATE TRIGGER require_enrollment_person_phone_before_write
BEFORE INSERT OR UPDATE OF person_id, organization_id ON enrollments
FOR EACH ROW
EXECUTE FUNCTION require_enrollment_person_phone();

CREATE OR REPLACE FUNCTION prevent_open_enrollment_phone_removal()
RETURNS TRIGGER AS $$
BEGIN
    IF NEW.profile_status <> 'anonymized'
       AND (NEW.phone_e164 IS NULL OR btrim(NEW.phone_e164) = '')
       AND OLD.phone_e164 IS NOT NULL
       AND EXISTS (
           SELECT 1
             FROM enrollments
            WHERE organization_id = OLD.organization_id
              AND person_id = OLD.id
              AND lifecycle_status = 'active'
       ) THEN
        RAISE EXCEPTION USING
            ERRCODE = '23514',
            CONSTRAINT = 'persons_open_enrollment_phone_required',
            MESSAGE = 'the WhatsApp phone cannot be removed while an enrollment is open';
    END IF;

    RETURN NEW;
END;
$$ LANGUAGE plpgsql;

CREATE TRIGGER prevent_open_enrollment_phone_removal_before_update
BEFORE UPDATE OF phone_e164, profile_status ON persons
FOR EACH ROW
EXECUTE FUNCTION prevent_open_enrollment_phone_removal();

DO $$
DECLARE
    invalid_open_enrollments BIGINT;
BEGIN
    SELECT COUNT(*)
      INTO invalid_open_enrollments
      FROM enrollments e
      JOIN persons p
        ON p.id = e.person_id
       AND p.organization_id = e.organization_id
     WHERE e.lifecycle_status = 'active'
       AND (p.phone_e164 IS NULL OR btrim(p.phone_e164) = '');

    IF invalid_open_enrollments > 0 THEN
        RAISE EXCEPTION USING
            ERRCODE = '23514',
            CONSTRAINT = 'open_enrollments_person_phone_required',
            MESSAGE = format('%s open enrollment(s) have no WhatsApp phone', invalid_open_enrollments),
            HINT = 'Register a valid phone for every person with an open enrollment, then run the migration again.';
    END IF;
END;
$$;
