package main

import (
	"context"
	"log"
	"os"

	"github.com/cloudwego/hertz/pkg/app"
	"github.com/cloudwego/hertz/pkg/app/server"
	"github.com/cloudwego/hertz/pkg/common/utils"
	"github.com/cloudwego/hertz/pkg/protocol/consts"
	"github.com/trae/bookinfo/pkg/otel"
	"go.opentelemetry.io/otel/baggage"
)

func main() {
	// 1. 初始化 Tracer (确保配置了全局 Propagator)(
	shutdown, err := otel.InitTracer("ratings")
	if err != nil {
		log.Fatalf("Failed to initialize tracer: %v", err)
	}
	defer shutdown(context.Background())

	// 2. 创建 Hertz 服务
	h := server.Default(
		server.WithHostPorts("0.0.0.0:9080"),
	)

	// 3. 使用增强版中间件 (确保能解析出上游透传的 pr_id)
	h.Use(otel.Middleware("ratings"))

	// 4. 定义路由
	h.GET("/ratings", func(c context.Context, ctx *app.RequestContext) {
		id := "test"
		// 获取 hostname
		hostname, _ := os.Hostname()
		// --- 隔离验证逻辑 ---
		// 从 Context 中提取 Baggage，查看当前请求是否属于某个 PR
		b := baggage.FromContext(c)
		prID := b.Member("pr_id").Value()

		// 如果是在测试 PR 流量，我们可以在响应中添加标识，方便调试验证
		stars := 4
		if prID != "" {
			log.Printf("[PR-%s] Processing ratings for product: %s", prID, id)
			// 甚至可以根据 PR ID 模拟不同的业务行为
		}

		rating := map[string]interface{}{
			"id":                id,
			"stars":             stars,
			"color":             "black",
			"hostname":          hostname,
			"active_pr_context": prID, // 将识别到的 PR ID 返回，方便 E2E 测试校验
		}

		ctx.JSON(consts.StatusOK, utils.H{
			"ratings": rating,
		})
	})

	log.Println("Ratings service started on :9080")
	if err := h.Run(); err != nil {
		log.Fatalf("Failed to run server: %v", err)
	}
}
