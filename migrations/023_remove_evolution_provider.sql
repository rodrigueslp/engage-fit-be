UPDATE whatsapp_settings
SET provider = 'twilio',
    base_url = CASE WHEN base_url ILIKE '%evolution%' OR base_url ILIKE 'mock://%' THEN '' ELSE base_url END,
    instance_name = CASE WHEN base_url ILIKE '%evolution%' OR base_url ILIKE 'mock://%' THEN '' ELSE instance_name END,
    enabled = CASE WHEN provider = 'evolution' THEN false ELSE enabled END,
    updated_at = NOW()
WHERE provider = 'evolution'
   OR provider IS NULL
   OR provider = '';
