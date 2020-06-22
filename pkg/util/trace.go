package util

import (
	"context"
	"github.com/opentracing/opentracing-go"
)

func CreateParentSpan (ctx context.Context, name string){
	tracer := opentracing.GlobalTracer()
	span := tracer.StartSpan(name)
	ctx = context.WithValue(ctx,tracer,span)
	defer span.Finish()
}

func CreateChildSpan(ctx context.Context, name string){
	if parent, ok := ctx.Value(opentracing.GlobalTracer()).(opentracing.Span); ok {
		tracer := opentracing.GlobalTracer()

		child := tracer.StartSpan(
			name,
			opentracing.ChildOf(parent.Context()))
		defer child.Finish()
	}
}
