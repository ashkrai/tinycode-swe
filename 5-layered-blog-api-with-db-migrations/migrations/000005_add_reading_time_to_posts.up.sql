-- Practice: adding a column with a default and backfill
ALTER TABLE posts
    ADD COLUMN reading_time_minutes SMALLINT NOT NULL DEFAULT 1;

-- Backfill: estimate from word count (avg 200 words/min)
UPDATE posts
   SET reading_time_minutes = GREATEST(
           1,
           (array_length(string_to_array(trim(body), ' '), 1) / 200)::SMALLINT
       );

COMMENT ON COLUMN posts.reading_time_minutes
    IS 'Estimated reading time in minutes (body word count / 200)';
