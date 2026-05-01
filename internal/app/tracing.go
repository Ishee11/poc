package app

import (
	"context"
	"log/slog"
	"strings"

	"go.opentelemetry.io/otel"
	"go.opentelemetry.io/otel/exporters/otlp/otlptrace/otlptracehttp"
	"go.opentelemetry.io/otel/propagation"
	"go.opentelemetry.io/otel/sdk/resource"
	sdktrace "go.opentelemetry.io/otel/sdk/trace"
	semconv "go.opentelemetry.io/otel/semconv/v1.34.0"
)

func setupTracing(ctx context.Context, cfg TracingConfig) (func(context.Context) error, error) {
	if cfg.OTLPEndpoint == "" {
		return func(context.Context) error { return nil }, nil
	}

	var opts []otlptracehttp.Option
	if strings.HasPrefix(cfg.OTLPEndpoint, "http://") || strings.HasPrefix(cfg.OTLPEndpoint, "https://") {
		opts = append(opts, otlptracehttp.WithEndpointURL(cfg.OTLPEndpoint))
	} else {
		opts = append(opts, otlptracehttp.WithEndpoint(cfg.OTLPEndpoint))
	}
	if cfg.OTLPInsecure {
		opts = append(opts, otlptracehttp.WithInsecure())
	}

	exporter, err := otlptracehttp.New(ctx, opts...)
	if err != nil {
		return nil, err
	}

	res, err := resource.Merge(
		resource.Default(),
		resource.NewWithAttributes(
			semconv.SchemaURL,
			semconv.ServiceName(cfg.ServiceName),
		),
	)
	if err != nil {
		return nil, err
	}

	provider := sdktrace.NewTracerProvider(
		sdktrace.WithBatcher(exporter),
		sdktrace.WithResource(res),
	)
	otel.SetTracerProvider(provider)
	otel.SetTextMapPropagator(propagation.NewCompositeTextMapPropagator(
		propagation.TraceContext{},
		propagation.Baggage{},
	))

	slog.Info("tracing_enabled", "service_name", cfg.ServiceName, "otlp_endpoint", cfg.OTLPEndpoint)
	return provider.Shutdown, nil
}
