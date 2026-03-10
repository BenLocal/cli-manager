ALTER TABLE session_record ADD COLUMN process TEXT NOT NULL DEFAULT 'bash';

UPDATE session_record
SET process = CASE
    WHEN process = '' THEN 'bash'
    ELSE process
END;
