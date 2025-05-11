package util

import (
	"math"

	"github.com/sweetrpg/api-core.go/constants"
	"github.com/sweetrpg/common.go/logging"
	dbconstants "github.com/sweetrpg/db.go/constants"
	options "go.jtlabs.io/query"
)

type Sort struct {
	Field string
	Order int
}

type Filter struct {
	Field     string
	Operation *string
	Value     []string
}

type Projection struct {
	Field     string
	Inclusion bool
}

type QueryParams struct {
	Start      int64
	Limit      int
	Sort       []Sort
	Filter     []Filter
	Projection []Projection
}

func GetQueryParams(query string) QueryParams {
	logging.Logger.Debug("parsing query string", "query", query)

	opt, _ := options.FromQuerystring(query)
	logging.Logger.Debug("options from query string", "opt", opt)

	var sortFields []Sort // bson.D
	for _, v := range opt.Sort {
		logging.Logger.Debug("sort field", "v", v)
		sortFields = append(sortFields, Sort{v, 1} /*bson.E{v, 1}*/)
	}

	var filters []Filter // bson.D
	for k, v := range opt.Filter {
		logging.Logger.Debug("filter", "k", k, "v", v)
		filters = append(filters, Filter{k, nil, v} /*bson.E{k, v}*/)
	}

	var proj []Projection // bson.D
	for _, v := range opt.Fields {
		logging.Logger.Debug("projection", "v", v)
		proj = append(proj, Projection{v, true} /*bson.E{v, 1}*/)
	}

	limit := opt.Page[constants.PageLimitOption]
	if limit == 0 {
		limit = dbconstants.QueryDefaultSize
	}

	params := QueryParams{
		Start:      int64(math.Max(0, float64(opt.Page[constants.PageStartOption]))),
		Limit:      int(math.Max(1, math.Min(float64(dbconstants.QueryMaxSize), float64(limit)))),
		Sort:       sortFields,
		Filter:     filters,
		Projection: proj,
	}

	logging.Logger.Debug("final params", "params", params)
	return params
}
