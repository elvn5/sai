-- +goose Up
-- Outings merged into tours: the winter series no longer points readers to
-- the «Выходы» section.
UPDATE blog_posts
SET content = replace(content,
        'Ближайшие выходы и туры по этим направлениям ищите в разделах «Выходы» и «Туры»',
        'Ближайшие туры по этим направлениям ищите в разделе «Туры»'),
    updated_at = now()
WHERE locale = 'ru' AND slug = 'zimnie-pohody-populyarnye-napravleniya'
  AND content LIKE '%в разделах «Выходы» и «Туры»%';

-- +goose Down
SELECT 1;
