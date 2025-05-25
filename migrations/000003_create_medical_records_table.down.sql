DROP TRIGGER IF EXISTS set_medical_records_updated_at ON medical_records;
DROP TABLE IF EXISTS medical_records;

-- The trigger function trigger_set_timestamp is potentially used by other tables
-- (like patients). It should only be dropped if no other table needs it.
-- For simplicity in this migration's down script, we are not dropping it here.
-- If it were exclusively for medical_records, you'd add:
-- DROP FUNCTION IF EXISTS trigger_set_timestamp();
-- However, since 000001_create_patients_table.up.sql also creates/uses it,
-- its down script (000001_create_patients_table.down.sql) is responsible for dropping it.
