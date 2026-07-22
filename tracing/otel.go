package tracing

import (
	"context"
	"fmt"

	"github.com/sweetrpg/api-core.go/util"
	"go.opentelemetry.io/otel"
	"go.opentelemetry.io/otel/attribute"
	oteltrace "go.opentelemetry.io/otel/trace"
)

func BuildSpanWithParams(c context.Context, tracerName string, spanName string, params util.QueryParams) oteltrace.Span {

	pageItems := make([]string, 0, 2)
	pageItems = append(pageItems, fmt.Sprintf("start=%d", params.Start))
	pageItems = append(pageItems, fmt.Sprintf("limit=%d", params.Limit))

	sortItems := make([]string, 0, len(params.Sort))
	for _, v := range params.Sort {
		sortItems = append(sortItems, fmt.Sprintf("%s=%d", v.Field, v.Order))
	}

	projItems := make([]string, 0, len(params.Projection))
	for _, v := range params.Projection {
		projItems = append(projItems, fmt.Sprintf("%s=%v", v.Field, v.Inclusion))
	}

	filterItems := make([]string, 0, len(params.Filter))
	for _, v := range params.Filter {
		filterItems = append(filterItems, fmt.Sprintf("%s-%v-%v", v.Field, v.Operation, v.Value))
	}

	_, span := otel.Tracer(tracerName).Start(c, spanName,
		oteltrace.WithAttributes(attribute.StringSlice("fields", projItems)),
		oteltrace.WithAttributes(attribute.StringSlice("sort", sortItems)),
		oteltrace.WithAttributes(attribute.StringSlice("filter", filterItems)),
		oteltrace.WithAttributes(attribute.StringSlice("page", pageItems)))

	return span
}
