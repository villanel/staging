package otel

import (
	"context"
	"fmt"
	"log"
	"os"
	"time"

	"go.opentelemetry.io/otel"
	"go.opentelemetry.io/otel/attribute"
	"go.opentelemetry.io/otel/exporters/otlp/otlptrace/otlptracegrpc"
	"go.opentelemetry.io/otel/propagation"
	"go.opentelemetry.io/otel/sdk/resource"
	"go.opentelemetry.io/otel/sdk/trace"
	semconv "go.opentelemetry.io/otel/semconv/v1.21.0"
)

func InitTracer(serviceName string) (func(context.Context) error, error) {
	// 1. 从环境变量获取 Collector 地址，默认为 localhost
	endpoint := os.Getenv("OTEL_EXPORTER_OTLP_ENDPOINT")
	if endpoint == "" {
		endpoint = "localhost:4317"
	}
	log.Printf("service PR_ID:%s", os.Getenv("PR_ID"))
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	// 2. 创建导出器 (增加超时控制)
	exporter, err := otlptracegrpc.New(ctx,
		otlptracegrpc.WithInsecure(),
		otlptracegrpc.WithEndpoint(endpoint),
	)
	if err != nil {
		return nil, fmt.Errorf("failed to create exporter: %w", err)
	}

	// 3. 构建 Resource，注入更多元数据（如 PR 标识）
	attrs := []attribute.KeyValue{
		semconv.ServiceNameKey.String(serviceName),
	}
	if prID := os.Getenv("PR_ID"); prID != "" {
		attrs = append(attrs, attribute.String("service.pr_id", prID))
	}

	res, err := resource.New(context.Background(),
		resource.WithAttributes(attrs...),
	)
	if err != nil {
		return nil, fmt.Errorf("failed to create resource: %w", err)
	}

	// 4. 初始化 TracerProvider
	tp := trace.NewTracerProvider(
		trace.WithBatcher(exporter),
		trace.WithResource(res),
	)

	// 5. 设置全局变量
	otel.SetTracerProvider(tp)

	// 关键点：必须包含 Baggage，否则自定义路由 Header 无法跨服务传递
	otel.SetTextMapPropagator(propagation.NewCompositeTextMapPropagator(
		propagation.TraceContext{},
		propagation.Baggage{},
	))

	return tp.Shutdown, nil
}
