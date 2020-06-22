package util

import (
	"context"
	"github.com/opentracing/opentracing-go"
)

type SMTrace struct {
	childSpan opentracing.Span
}

func New(span opentracing.Span) SMTrace{
	return SMTrace {
		childSpan: span,
	}
}

func (smt SMTrace) FinishSpan() {
	if smt.childSpan != nil {
		smt.childSpan.Finish()
	}
}

func (smt SMTrace) GetSpan() opentracing.Span {
	return smt.childSpan
}

func CreateParentSpan (ctx context.Context, name string) {
	tracer := opentracing.GlobalTracer()
	span := tracer.StartSpan(name)
	ctx = context.WithValue(ctx,tracer,span)
}

func CreateChildSpan(ctx context.Context, name string) SMTrace {
	if parent, ok := ctx.Value(opentracing.GlobalTracer()).(opentracing.Span); ok {
		tracer := opentracing.GlobalTracer()

		return New(tracer.StartSpan(
			name,
			opentracing.ChildOf(parent.Context())))


	}

	return SMTrace{}
}
