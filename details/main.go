package main

import (
	"context"
	"log"

	"github.com/cloudwego/hertz/pkg/app"
	"github.com/cloudwego/hertz/pkg/app/server"
	"github.com/cloudwego/hertz/pkg/common/utils"
	"github.com/cloudwego/hertz/pkg/protocol/consts"
	"github.com/trae/bookinfo/pkg/otel"
)

func main() {
	// Initialize OpenTelemetry tracer
	shutdown, err := otel.InitTracer("details")
	if err != nil {
		log.Fatalf("Failed to initialize tracer: %v", err)
	}
	defer shutdown(context.Background())

	// Create Hertz server
	h := server.Default(
		server.WithHostPorts("0.0.0.0:9081"),
	)

	// Use OpenTelemetry middleware
	h.Use(otel.Middleware("details"))

	// Define route
	h.GET("/details/:id", func(c context.Context, ctx *app.RequestContext) {
		id := ctx.Param("id")
		book := map[string]interface{}{
			"id":          id,
			"author":      "William Shakespeare",
			"year":        1595,
			"type":        "paperback",
			"pages":       200,
			"publisher":   "PublisherA",
			"language":    "English",
			"ISBN-10":     "1234567890",
			"ISBN-13":     "123-1234567890",
		}
		ctx.JSON(consts.StatusOK, utils.H{
			"details": book,
		})
	})

	log.Println("Details service started on :9081")
	if err := h.Run(); err != nil {
		log.Fatalf("Failed to run server: %v", err)
	}
}