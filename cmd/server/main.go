package main

import (
	"log"

	"github.com/gin-gonic/gin"
	"github.com/thebowwman/delitrack/internals/api"
)

func main() {
	r := gin.Default()
	api.RegisterRoutes(r)

	addr := ":8081"
	log.Printf("listening on %s", addr)
	if err := r.Run(addr); err != nil {
		log.Fatal(err)
	}
}
