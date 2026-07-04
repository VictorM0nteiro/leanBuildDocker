package main

import (
	"embed"
	"net/http"
)

// templatesFS bakes the templates into the binary, so the runtime image must
// NOT copy templates/ — the analyzer should classify it as embedded.
//
//go:embed templates/*
var templatesFS embed.FS

func main() {
	http.Handle("/", http.FileServer(http.FS(templatesFS)))
	_ = http.ListenAndServe(":8000", nil)
}
