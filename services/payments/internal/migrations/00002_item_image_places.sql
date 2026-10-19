-- +goose Up
-- The booked item's picture (the tour's cover), for the customer's bookings.
ALTER TABLE orders ADD COLUMN item_image_url text;
-- Places taken per item: paid orders and pending ones still on hold.
CREATE INDEX orders_item_idx ON orders (item_id) WHERE status IN ('paid', 'pending');

-- +goose Down
DROP INDEX orders_item_idx;
ALTER TABLE orders DROP COLUMN item_image_url;
