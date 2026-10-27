-- +goose Up
ALTER TABLE trails ADD COLUMN markers jsonb NOT NULL DEFAULT '[]'::jsonb;
ALTER TABLE trails ADD COLUMN checklist_url text;

-- +goose Down
ALTER TABLE trails DROP COLUMN checklist_url;
ALTER TABLE trails DROP COLUMN markers;
