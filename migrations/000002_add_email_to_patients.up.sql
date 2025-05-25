ALTER TABLE patients
ADD COLUMN email VARCHAR(255) UNIQUE;

-- Note: If existing rows might violate the UNIQUE constraint,
-- you would need a more complex migration, possibly involving:
-- 1. Add the column without the UNIQUE constraint.
-- 2. Populate/clean up email data for existing rows.
-- 3. Add the UNIQUE constraint.
-- For this exercise, we assume it's a new column and new data will be unique,
-- or the table is empty.
