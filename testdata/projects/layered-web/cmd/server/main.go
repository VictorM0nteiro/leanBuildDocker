package main

import (
	"log"
	"os"

	"github.com/gin-gonic/gin"

	"example.com/layered-web/internal/handler"
)

// version is stamped at build time via -ldflags -X main.version=...
var version = "dev"

func main() {
	r := gin.Default()
	r.LoadHTMLGlob("templates/*")
	r.Static("/static", "./static")
	handler.Register(r)

	addr := ":9090"
	if p := os.Getenv("PORT"); p != "" {
		addr = ":" + p
	}
	log.Printf("layered-web %s listening on %s", version, addr)
	if err := r.Run(addr); err != nil {
		log.Fatal(err)
	}
}
