package api

import "context"

// CabinetRequest uses the same short-lived user session as the bot's other
// sections. Paths come only from the cabinet's server-side resource registry,
// never from Telegram callback data or user text. Backend ownership checks are
// authoritative for every read and write.
func (c *Client) CabinetRequest(ctx context.Context, userID, method, path string, body, out any) error {
	return c.do(ctx, method, path, userID, body, out)
}
