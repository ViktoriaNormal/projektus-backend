ALTER TABLE users
    DROP COLUMN IF EXISTS on_vacation,
    DROP COLUMN IF EXISTS is_sick,
    DROP COLUMN IF EXISTS alternative_contact_channel,
    DROP COLUMN IF EXISTS alternative_contact_info;
