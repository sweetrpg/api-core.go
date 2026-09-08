package util

import (
	"strings"

	"github.com/sweetrpg/common.go/logging"

	"go.mongodb.org/mongo-driver/bson"
)

func ConvertQueryParams(params QueryParams) (filter bson.D, sort bson.D, projection bson.D) {
	logging.Logger.Debug("Converting query params to BSON", "params", params)

	for _, v := range params.Filter {
		logging.Logger.Debug("Processing filter", "field", v.Field, "operation", v.Operation)
		operation := "$eq"
		if v.Operation != nil {
			operation = *v.Operation
		}

		var clause bson.D
		if operation == "$regex" {
			// contains: a case-insensitive substring match. The client value is the pattern;
			// $options "i" is fixed here, never client-supplied.
			clause = bson.D{{Key: "$regex", Value: regexPattern(v.Value)}, {Key: "$options", Value: "i"}}
		} else {
			clause = bson.D{{Key: operation, Value: v.Value}}
		}
		filter = append(filter, bson.E{Key: v.Field, Value: clause})
	}

	for _, v := range params.Sort {
		logging.Logger.Debug("Processing sort", "field", v.Field, "order", v.Order)
		sort = append(sort, bson.E{Key: v.Field, Value: v.Order})
	}

	for _, v := range params.Projection {
		logging.Logger.Debug("Processing projection", "field", v.Field, "inclusion", v.Inclusion)
		inclusion := 0
		if v.Inclusion {
			inclusion = 1
		}
		projection = append(projection, bson.E{Key: v.Field, Value: inclusion})
	}

	return
}

// regexPattern reduces a filter's value list to a single $regex pattern string. A contains
// filter normally carries exactly one value; multiple values (filter[f][contains]=a,b) become a
// regex alternation so the match still means "any of".
func regexPattern(values []string) string {
	if len(values) == 1 {
		return values[0]
	}
	return strings.Join(values, "|")
}
