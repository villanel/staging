package otel

import (
	"context"
	"time"

	"github.com/cloudwego/hertz/pkg/app"
	"github.com/cloudwego/hertz/pkg/protocol" // 修复：导入 protocol 包
	"github.com/cloudwego/hertz/pkg/protocol/consts"
	"go.opentelemetry.io/otel"
	"go.opentelemetry.io/otel/attribute"
	"go.opentelemetry.io/otel/baggage"
	"go.opentelemetry.io/otel/codes"
	"go.opentelemetry.io/otel/propagation" // 修复：确保下面的代码使用了它
	"go.opentelemetry.io/otel/trace"
)

// hertzHeaderCarrier 适配器
// 显式声明实现 propagation.TextMapCarrier 接口，以确保包被使用并验证正确性
var _ propagation.TextMapCarrier = (*hertzHeaderCarrier)(nil)

type hertzHeaderCarrier struct {
	header *protocol.RequestHeader // 修复：使用 protocol.RequestHeader
}

func (c *hertzHeaderCarrier) Get(key string) string {
	return string(c.header.Get(key))
}

func (c *hertzHeaderCarrier) Set(key string, value string) {
	c.header.Set(key, value)
}

func (c *hertzHeaderCarrier) Keys() []string {
	keys := make([]string, 0, c.header.Len())
	c.header.VisitAll(func(k, v []byte) {
		keys = append(keys, string(k))
	})
	return keys
}

// Middleware 为 OpenTelemetry 增强版中间件
func Middleware(serviceName string) app.HandlerFunc {
	return func(c context.Context, ctx *app.RequestContext) {
		// 1. 提取上下文
		propagator := otel.GetTextMapPropagator()
		// 注意这里传指针 &ctx.Request.Header
		carrier := &hertzHeaderCarrier{header: &ctx.Request.Header}
		c = propagator.Extract(c, carrier)

		// 2. 处理 PR 染色标识
		prID := string(ctx.GetHeader("x-pr-id"))
		// fmt.Printf("Step 1:  prID: %s\n", prID)
		if prID != "" {

			m, err := baggage.NewMember("pr_id", prID)
			if err != nil {
				// 如果这里报错，你会看到具体的字符限制原因
			} else {
				b, _ := baggage.New(m)
				c = baggage.ContextWithBaggage(c, b)

				// // 验证当前 Context 是否存入成功
				// d := baggage.FromContext(c)
				// val := d.Member("pr_id").Value()
				// fmt.Printf("Step 3: Baggage in Context: %s\n", val)
			}
		}
		// 3. 开始 Trace
		tracer := otel.Tracer("hertz")
		method := string(ctx.Request.Method())
		path := string(ctx.Request.URI().Path())

		sctx, span := tracer.Start(
			c,
			method+" "+path,
			trace.WithSpanKind(trace.SpanKindServer),
			trace.WithAttributes(
				attribute.String("service.name", serviceName),
				attribute.String("http.method", method),
				attribute.String("http.path", path),
			),
		)
		defer span.End()

		// 4. 继续链式调用
		start := time.Now()
		ctx.Next(sctx)
		duration := time.Since(start)

		// 5. 记录响应
		statusCode := ctx.Response.StatusCode()
		span.SetAttributes(
			attribute.Int("http.status_code", statusCode),
			attribute.Float64("http.duration_ms", float64(duration.Milliseconds())),
		)

		if statusCode >= consts.StatusInternalServerError {
			span.SetStatus(codes.Error, "internal_server_error")
		}
	}
}
