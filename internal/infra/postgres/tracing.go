package postgres

import (
	"context"
	"strings"

	"github.com/jackc/pgx/v5"
	"go.opentelemetry.io/otel"
	"go.opentelemetry.io/otel/attribute"
	"go.opentelemetry.io/otel/codes"
	"go.opentelemetry.io/otel/trace"
)

type QueryTracer struct {
	tracer trace.Tracer
}

func NewQueryTracer() *QueryTracer {
	return &QueryTracer{tracer: otel.Tracer("github.com/ishee11/poc/internal/infra/postgres")}
}

func (t *QueryTracer) TraceQueryStart(ctx context.Context, _ *pgx.Conn, data pgx.TraceQueryStartData) context.Context {
	operation := queryOperation(data.SQL)
	ctx, span := t.tracer.Start(ctx, "postgres."+operation, trace.WithSpanKind(trace.SpanKindClient))
	span.SetAttributes(
		attribute.String("db.system.name", "postgresql"),
		attribute.String("db.operation.name", operation),
		attribute.String("db.query.text", compactSQL(data.SQL)),
	)
	return ctx
}

func (t *QueryTracer) TraceQueryEnd(ctx context.Context, _ *pgx.Conn, data pgx.TraceQueryEndData) {
	span := trace.SpanFromContext(ctx)
	if data.Err != nil {
		span.RecordError(data.Err)
		span.SetStatus(codes.Error, data.Err.Error())
	}
	if data.CommandTag.String() != "" {
		span.SetAttributes(attribute.String("db.response.status_code", data.CommandTag.String()))
	}
	span.End()
}

func queryOperation(sql string) string {
	fields := strings.Fields(sql)
	if len(fields) == 0 {
		return "query"
	}
	return strings.ToUpper(fields[0])
}

func compactSQL(sql string) string {
	return strings.Join(strings.Fields(sql), " ")
}
