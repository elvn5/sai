// Package proxy forwards requests to the backend services.
package proxy

import (
	"context"
	"encoding/json"
	"errors"
	"log/slog"
	"net/http"
	"net/http/httputil"
	"net/url"
	"strings"
	"time"

	"github.com/gin-gonic/gin"

	"github.com/elvn5/sai/services/pkg/httperr"
)

// New returns a handler that forwards the request unchanged (method, path,
// query, body, cookies) to target and streams the response back.
//
// CORS belongs to the gateway: any CORS headers the upstream sets are
// dropped so browsers don't see them twice.
func New(target *url.URL, timeout time.Duration, log *slog.Logger) gin.HandlerFunc {
	transport := http.DefaultTransport.(*http.Transport).Clone()
	transport.ResponseHeaderTimeout = timeout

	rp := &httputil.ReverseProxy{
		Transport: transport,
		Rewrite: func(pr *httputil.ProxyRequest) {
			pr.SetURL(target)
			// Keep the public Host, and extend rather than replace any
			// forwarding chain set by nginx in front of the gateway.
			pr.Out.Host = pr.In.Host
			if prior := pr.In.Header.Values("X-Forwarded-For"); len(prior) > 0 {
				pr.Out.Header["X-Forwarded-For"] = prior
			}
			pr.SetXForwarded()
			if proto := pr.In.Header.Get("X-Forwarded-Proto"); proto != "" {
				pr.Out.Header.Set("X-Forwarded-Proto", proto)
			}
		},
		ModifyResponse: func(res *http.Response) error {
			stripCORS(res.Header)
			return nil
		},
		ErrorHandler: func(w http.ResponseWriter, r *http.Request, err error) {
			if errors.Is(err, context.Canceled) {
				return // the client went away; nothing to answer
			}
			log.ErrorContext(r.Context(), "upstream request failed",
				slog.String("upstream", target.String()),
				slog.String("path", r.URL.Path),
				slog.Any("error", err),
			)
			w.Header().Set("Content-Type", "application/json; charset=utf-8")
			w.WriteHeader(http.StatusBadGateway)
			_ = json.NewEncoder(w).Encode(httperr.New(http.StatusBadGateway, "Upstream service unavailable"))
		},
	}

	return func(c *gin.Context) {
		rp.ServeHTTP(c.Writer, c.Request)
	}
}

func stripCORS(header http.Header) {
	for name := range header {
		if strings.HasPrefix(name, "Access-Control-") {
			header.Del(name)
		}
	}

	var vary []string
	for _, value := range header.Values("Vary") {
		for _, token := range strings.Split(value, ",") {
			token = strings.TrimSpace(token)
			if token != "" && !strings.EqualFold(token, "Origin") {
				vary = append(vary, token)
			}
		}
	}
	header.Del("Vary")
	if len(vary) > 0 {
		header.Set("Vary", strings.Join(vary, ", "))
	}
}
