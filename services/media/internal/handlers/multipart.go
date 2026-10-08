package handlers

import (
	"errors"
	"io"
	"mime/multipart"
	"net/http"

	"github.com/gin-gonic/gin"

	"github.com/elvn5/sai/services/pkg/httperr"
)

const fileField = "file"

// uploadError is a client-facing failure, worded like NestJS + multer.
type uploadError struct {
	status  int
	message string
}

func (e *uploadError) Error() string { return e.message }

var (
	errNoFile          = &uploadError{http.StatusBadRequest, "No file uploaded"}
	errUnexpectedField = &uploadError{http.StatusBadRequest, "Unexpected field"}
	errFileTooLarge    = &uploadError{http.StatusRequestEntityTooLarge, "File too large"}
	errMalformedForm   = &uploadError{http.StatusBadRequest, "Malformed multipart body"}
)

// receiveFile reads a multipart request the way NestJS's
// FileInterceptor("file") did: exactly one file in the "file" field (any
// other file part is "Unexpected field"), text fields ignored, filter run
// on the part headers before any bytes are read, and maxBytes enforced
// while streaming. store consumes the file and returns the byte count.
func receiveFile(
	c *gin.Context,
	maxBytes int64,
	filter func(part *multipart.Part) *uploadError,
	store func(part *multipart.Part, body io.Reader) (int64, error),
) (*multipart.Part, int64, error) {
	reader, err := c.Request.MultipartReader()
	if err != nil {
		// Not multipart: multer leaves the file undefined.
		return nil, 0, errNoFile
	}

	var accepted *multipart.Part
	var size int64
	for {
		part, err := reader.NextPart()
		if errors.Is(err, io.EOF) {
			break
		}
		if err != nil {
			return nil, 0, errMalformedForm
		}
		if part.FileName() == "" {
			continue // text field
		}
		if part.FormName() != fileField || accepted != nil {
			return nil, 0, errUnexpectedField
		}
		if rejected := filter(part); rejected != nil {
			return nil, 0, rejected
		}

		limited := &limitedReader{r: part, remaining: maxBytes}
		size, err = store(part, limited)
		if err != nil {
			return nil, 0, err
		}
		accepted = part
	}

	if accepted == nil {
		return nil, 0, errNoFile
	}
	return accepted, size, nil
}

// limitedReader fails with errFileTooLarge once more than the limit is read.
type limitedReader struct {
	r         io.Reader
	remaining int64
}

func (l *limitedReader) Read(p []byte) (int, error) {
	n, err := l.r.Read(p)
	l.remaining -= int64(n)
	if l.remaining < 0 {
		return n, errFileTooLarge
	}
	return n, err
}

// abortUpload writes an uploadError, or a generic 500 for anything else.
func abortUpload(c *gin.Context, err error) {
	var uerr *uploadError
	if errors.As(err, &uerr) {
		httperr.Abort(c, uerr.status, uerr.message)
		return
	}
	_ = c.Error(err)
	c.AbortWithStatusJSON(http.StatusInternalServerError, httperr.InternalServerError())
}
