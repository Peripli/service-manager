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

func CreateParentSpan (ctx context.Context, name string)  (context.Context,opentracing.Span) {
	tracer := opentracing.GlobalTracer()
	span := tracer.StartSpan(name)
	return context.WithValue(ctx,tracer,span),span
}

func CreateChildSpan(ctx context.Context, name string) (SMTrace,context.Context) {
	if parent, ok := ctx.Value(opentracing.GlobalTracer()).(opentracing.Span); ok {
		tracer := opentracing.GlobalTracer()
		newSpan := tracer.StartSpan(
			name,
			opentracing.ChildOf(parent.Context()))

		ctx = context.WithValue(ctx,tracer,newSpan)
		return New(newSpan),ctx
	}
	return SMTrace{},ctx
}