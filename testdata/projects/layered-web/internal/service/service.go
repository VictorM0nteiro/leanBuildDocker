package service

import (
	"io"
	"net/http"
	"time"

	"example.com/layered-web/internal/repository"
)

// Status makes an outbound HTTPS call (needs CA certs) and reads a timezone
// (needs tzdata) — the two runtime needs the analyzer must detect.
func Status() string {
	loc, _ := time.LoadLocation("America/Sao_Paulo")
	_ = loc

	resp, err := http.Get("https://example.com/health")
	if err != nil {
		return "degraded"
	}
	defer resp.Body.Close()
	_, _ = io.ReadAll(resp.Body)

	repo := repository.New()
	repo.Set("last_check", time.Now().String())
	return "ok"
}
