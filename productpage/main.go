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

// Details 和 Reviews 结构体定义保持不变...
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
	} `json:"details"`
}

type Reviews struct {
	Reviews struct {
		ID      string `json:"id"`
		Reviews []struct {
			Reviewer string `json:"reviewer"`
			Text     string `json:"text"`
			Rating   int    `json:"rating"`
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

		// --- 调用 Details 服务 ---
		_, detailsBody, err := hc.Get(c, nil, fmt.Sprintf("%s/details", detailsURL ))
		if err != nil {
			ctx.JSON(consts.StatusInternalServerError, utils.H{"error": "details service: " + err.Error()})
			return
		}
		var details Details
		json.Unmarshal(detailsBody, &details)

		// --- 调用 Reviews 服务 ---
		// 修正点 1: 使用新变量 reviewsBody，这样 := 左侧就有新变量了
		_, reviewsBody, err := hc.Get(c, nil, fmt.Sprintf("%s/reviews", reviewsURL))
		if err != nil {
			ctx.JSON(consts.StatusInternalServerError, utils.H{"error": "reviews service: " + err.Error()})
			return
		}
		var reviews Reviews
		// 修正点 2: 使用正确的变量 reviewsBody 进行反序列化
		json.Unmarshal(reviewsBody, &reviews)

		// --- 聚合返回 ---
		ctx.JSON(consts.StatusOK, utils.H{
			"product": map[string]interface{}{
				"id":      "test",
				"title":   "The Comedy of Errors",
				"details": details.Details,
				"reviews": reviews.Reviews.Reviews,
			},
		})
	})

	h.Run()
}