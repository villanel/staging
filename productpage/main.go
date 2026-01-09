package main

import (
	"context"
	"encoding/json"
	"fmt"
	"log"
	"os"

	"github.com/cloudwego/hertz/pkg/app"
	"github.com/cloudwego/hertz/pkg/app/client"
	"github.com/cloudwego/hertz/pkg/app/server"
	"github.com/cloudwego/hertz/pkg/common/utils"
	"github.com/cloudwego/hertz/pkg/protocol/consts"
	"github.com/hertz-contrib/obs-opentelemetry/tracing"

	// 修复 1：为自定义包设置别名以避免冲突
	customOtel "github.com/trae/bookinfo/pkg/otel"
)

// Details 结构体添加 hostname 字段
type Details struct {
	Details struct {
		ID        string `json:"id"`
		Author    string `json:"author"`
		Year      int    `json:"year"`
		Type      string `json:"type"`
		Pages     int    `json:"pages"`
		Publisher string `json:"publisher"`
		Language  string `json:"language"`
		ISBN10    string `json:"ISBN-10"`
		ISBN13    string `json:"ISBN-13"`
		Hostname  string `json:"hostname,omitempty"`
	} `json:"details"`
}

// Reviews 结构体添加 hostname 相关字段
type Reviews struct {
	Reviews struct {
		ID       string `json:"id"`
		Hostname string `json:"hostname"`
		Reviews  []struct {
			Reviewer      string `json:"reviewer"`
			Text          string `json:"text"`
			Rating        int    `json:"rating"`
			RatingHostname string `json:"rating_hostname"`
		} `json:"reviews"`
	} `json:"reviews"`
}

func main() {
	detailsURL := os.Getenv("DETAILS_SERVICE_URL")
	if detailsURL == "" {
		detailsURL = "http://localhost:9081"
	}

	reviewsURL := os.Getenv("REVIEWS_SERVICE_URL")
	if reviewsURL == "" {
		reviewsURL = "http://localhost:9082"
	}

	shutdown, err := customOtel.InitTracer("productpage")
	if err != nil {
		log.Fatalf("Failed to initialize tracer: %v", err)
	}
	defer shutdown(context.Background())

	h := server.Default(server.WithHostPorts("0.0.0.0:9083"))
	h.Use(customOtel.Middleware("productpage"))

	hc, _ := client.NewClient()
	hc.Use(tracing.ClientMiddleware())

	h.GET("/productpage", func(c context.Context, ctx *app.RequestContext) {
		// Get hostname
		hostname, _ := os.Hostname()

		// --- 调用 Details 服务 ---
		_, detailsBody, err := hc.Get(c, nil, fmt.Sprintf("%s/details", detailsURL))
		if err != nil {
			ctx.JSON(consts.StatusInternalServerError, utils.H{"error": "details service: " + err.Error()})
			return
		}
		var details Details
		json.Unmarshal(detailsBody, &details)

		// --- 调用 Reviews 服务 ---
		_, reviewsBody, err := hc.Get(c, nil, fmt.Sprintf("%s/reviews", reviewsURL))
		if err != nil {
			ctx.JSON(consts.StatusInternalServerError, utils.H{"error": "reviews service: " + err.Error()})
			return
		}
		var reviews Reviews
		json.Unmarshal(reviewsBody, &reviews)

		// --- 聚合返回 ---
		ctx.JSON(consts.StatusOK, utils.H{
			"product": map[string]interface{}{
				"id":              "test",
				"title":           "The Comedy of Errors",
				"hostname":        hostname,
				"details":         details.Details,
				"reviews":         reviews.Reviews,
			},
		})
	})

	h.Run()
}
