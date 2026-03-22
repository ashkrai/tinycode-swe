-- Practice: rolling back a column addition
ALTER TABLE posts
    DROP COLUMN IF EXISTS reading_time_minutes;
