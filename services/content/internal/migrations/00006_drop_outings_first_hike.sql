-- +goose Up
-- Outings merged into tours: the first-hike post no longer sends readers to
-- the «Выходы» section (00005 did the same for the winter series).
UPDATE blog_posts
SET content = replace(content,
        'присоединиться к открытому групповому выходу в разделе «Выходы» или выбрать однодневный тур',
        'присоединиться к групповому однодневному туру в разделе «Туры»'),
    updated_at = now()
WHERE locale = 'ru' AND slug = 'kak-vybrat-pervyj-odnodnevnyj-pohod-iz-bishkeka'
  AND content LIKE '%в разделе «Выходы»%';

-- +goose Down
SELECT 1;
