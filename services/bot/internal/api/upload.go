package api

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"mime/multipart"
	"net/http"
	"net/textproto"

	"github.com/elvn5/sai/services/pkg/auth"
)

// UploadImage stores Telegram photos in the existing media service; a bot file
// URL (which contains the bot token) must never be persisted or shown to users.
func (c *Client) UploadImage(ctx context.Context, userID string, data []byte) (string, error) {
	if len(data) == 0 || len(data) > 8<<20 {
		return "", fmt.Errorf("image exceeds upload limit")
	}
	var body bytes.Buffer
	writer := multipart.NewWriter(&body)
	header := textproto.MIMEHeader{}
	header.Set("Content-Disposition", `form-data; name="file"; filename="telegram.jpg"`)
	header.Set("Content-Type", http.DetectContentType(data))
	part, err := writer.CreatePart(header)
	if err != nil {
		return "", err
	}
	if _, err = part.Write(data); err != nil {
		return "", err
	}
	if err = writer.Close(); err != nil {
		return "", err
	}
	req, err := http.NewRequestWithContext(ctx, http.MethodPost, c.base+"/uploads/image", &body)
	if err != nil {
		return "", err
	}
	token, err := c.signer.UserToken(userID)
	if err != nil {
		return "", err
	}
	req.Header.Set("Cookie", auth.UserCookie+"="+token)
	req.Header.Set("Content-Type", writer.FormDataContentType())
	res, err := c.http.Do(req)
	if err != nil {
		return "", fmt.Errorf("image upload failed")
	}
	defer func() { _ = res.Body.Close() }()
	if res.StatusCode >= 400 {
		return "", &Error{Status: res.StatusCode, Message: "Image upload rejected"}
	}
	var out struct {
		URL string `json:"url"`
	}
	if err = json.NewDecoder(res.Body).Decode(&out); err != nil {
		return "", err
	}
	return out.URL, nil
}
