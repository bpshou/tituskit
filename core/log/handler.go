package log

import (
	"context"
	"log/slog"

	"go.opentelemetry.io/otel/trace"
)

var _ slog.Handler = (*Handler)(nil)

type Handler struct {
	slog.Handler
}

func NewHandler(handler slog.Handler) *Handler {
	return &Handler{Handler: handler}
}

func (h *Handler) Handle(ctx context.Context, record slog.Record) error {
	spanCtx := trace.SpanContextFromContext(ctx)
	if spanCtx.HasTraceID() {
		record.AddAttrs(slog.String("trace", spanCtx.TraceID().String()))
		if spanCtx.HasSpanID() {
			record.AddAttrs(slog.String("span", spanCtx.SpanID().String()))
		}
	}
	return h.Handler.Handle(ctx, record)
}

func (h *Handler) WithAttrs(attrs []slog.Attr) slog.Handler {
	return &Handler{Handler: h.Handler.WithAttrs(attrs)}
}

func (h *Handler) WithGroup(name string) slog.Handler {
	return &Handler{Handler: h.Handler.WithGroup(name)}
}
