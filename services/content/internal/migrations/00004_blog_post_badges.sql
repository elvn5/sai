-- +goose Up
-- What a post is about, shown as badges on its card: the difficulty of the
-- trips it describes ("any" for posts for every level), how long they take
-- (free text in the post's language) and the season. All optional.
ALTER TABLE blog_posts
    ADD COLUMN difficulty text CHECK (difficulty IN ('any', 'easy', 'moderate', 'hard', 'expert')),
    ADD COLUMN duration   text,
    ADD COLUMN season     text CHECK (season IN ('spring', 'summer', 'autumn', 'winter', 'all_year'));

-- The winter hiking series (00003).
UPDATE blog_posts AS p SET difficulty = v.difficulty, duration = v.duration, season = 'winter'
FROM (VALUES
    ('zimnie-pohody-snaryazhenie', 'any', 'От 1 дня'),
    ('zimnie-pohody-slozhnost-lokaciy', 'any', '2 часа — 4 дня'),
    ('zimnie-pohody-neslozhnye-lokacii', 'easy', '2–5 часов'),
    ('zimnie-pohody-samye-slozhnye-marshruty', 'hard', '1–4 дня'),
    ('zimnie-pohody-populyarnye-napravleniya', 'easy', '1–4 дня')
) AS v (slug, difficulty, duration)
WHERE p.locale = 'ru' AND p.slug = v.slug;

-- +goose Down
ALTER TABLE blog_posts DROP COLUMN season, DROP COLUMN duration, DROP COLUMN difficulty;
