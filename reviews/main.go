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
    
	// 修复 1：为本地 otel 包设置别名以避免命名冲突
	customOtel "github.com/trae/bookinfo/pkg/otel"
)

type Rating struct {
	Ratings struct {
		ID    string `json:"id"`
		Stars int    `json:"stars"`
		Color string `json:"color"`
	} `json:"ratings"`
}
func main() {
	// 1. 初始化 OTel (使用别名调用)
ratingsURL := os.Getenv("RATINGS_SERVICE_URL")
if ratingsURL == "" {
    ratingsURL = "http://localhost:9080"
}
	shutdown, err := customOtel.InitTracer("reviews")
	if err != nil {
		log.Fatalf("Failed to initialize tracer: %v", err)
	}
	defer shutdown(context.Background())

	h := server.Default(server.WithHostPorts("0.0.0.0:9082"))

	// 2. 服务端中间件：解析上游（ProductPage）传来的染色信息
	h.Use(customOtel.Middleware("reviews"))

	// 3. 客户端中间件：调用下游（Ratings）时自动透传 Context
	hc, _ := client.NewClient()
    
	// 修复 2：修正函数名为 ClientMiddleware()
	hc.Use(tracing.ClientMiddleware())

	h.GET("/reviews", func(c context.Context, ctx *app.RequestContext) {
		id := "test"

		// --- 调用 Ratings 服务 ---
		// 此处传入的 c 包含了从上游提取并经过中间件处理的 Baggage (pr_id)
		_, body, err := hc.Get(c, nil, fmt.Sprintf("%s/ratings", ratingsURL))
		if err != nil {
			ctx.JSON(consts.StatusInternalServerError, utils.H{"error": "ratings service: " + err.Error()})
			return
		}

		var rating Rating
		if err := json.Unmarshal(body, &rating); err != nil {
			ctx.JSON(consts.StatusInternalServerError, utils.H{"error": "json decode: " + err.Error()})
			return
		}

		// --- 聚合结果 ---
		reviews := map[string]interface{}{
			"id": id,
			"reviews": []map[string]interface{}{
				{
					"reviewer": "Reviewer1",
					"text":     "An extremely entertaining play by Shakespeare.",
					"rating":   rating.Ratings.Stars,
				},
				{
					"reviewer": "Reviewer2",
					"text":     "Absolutely fun and entertaining.",
					"rating":   rating.Ratings.Stars,
				},
			},
		}

		ctx.JSON(consts.StatusOK, utils.H{"reviews": reviews})
	})

	log.Println("Reviews service started on :9082")
	h.Run()
}