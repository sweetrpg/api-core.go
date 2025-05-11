package util

import (
	"unsafe"

	"github.com/sweetrpg/common.go/logging"

	"go.mongodb.org/mongo-driver/bson"
)

func ConvertQueryParams(params QueryParams) (filter bson.D, sort bson.D, projection bson.D) {
	logging.Logger.Debug("Converting query params to BSON", "params", params)

	for _, v := range params.Filter {
		logging.Logger.Debug("Processing filter", "field", v.Field, "operation", v.Operation)
		filter = append(filter, bson.E{Key: v.Field, Value: v.Operation})
	}

	for _, v := range params.Sort {
		logging.Logger.Debug("Processing sort", "field", v.Field, "order", v.Order)
		sort = append(sort, bson.E{Key: v.Field, Value: v.Order})
	}

	for _, v := range params.Projection {
		logging.Logger.Debug("Processing projection", "field", v.Field, "inclusion", v.Inclusion)
		projection = append(projection, bson.E{Key: v.Field, Value: *(*int)(unsafe.Pointer(&v.Inclusion)) & 1})
	}

	return
}
