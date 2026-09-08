package util

import (
	"fmt"
	"math"
	"strings"

	"github.com/sweetrpg/api-core.go/constants"
	"github.com/sweetrpg/common.go/logging"
	dbconstants "github.com/sweetrpg/mongodb.go/constants"
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

// filterOperators is the fixed allow-list of client-facing filter operators, each mapped to the
// one Mongo operator it is allowed to produce. A client-supplied operator segment is only ever a
// key into this map - it never reaches a bson.D as a literal operator string, which is what
// keeps a per-field operator syntax from also opening $where/$expr/arbitrary-operator injection.
var filterOperators = map[string]string{
	"eq":       "$eq",
	"contains": "$regex",
	"in":       "$in",
}

// parseFilterKey splits a raw filter key from the querystring parser into its field name and, if
// present, its Mongo operator. The bare form "field" (from filter[field]=value) returns a nil
// operator, preserving the historical default-to-$eq behavior. The operator form "field][seg"
// (from filter[field][seg]=value) maps seg through filterOperators; an unrecognized seg is an
// error, surfaced by callers as a 400 rather than falling back to $eq. More than one operator
// segment is malformed.
func parseFilterKey(key string) (field string, operation *string, err error) {
	parts := strings.Split(key, "][")
	switch len(parts) {
	case 1:
		return parts[0], nil, nil
	case 2:
		mongoOp, ok := filterOperators[parts[1]]
		if !ok {
			return "", nil, fmt.Errorf("unsupported filter operator %q on field %q", parts[1], parts[0])
		}
		return parts[0], &mongoOp, nil
	default:
		return "", nil, fmt.Errorf("malformed filter key %q", key)
	}
}

// GetQueryParams parses a raw querystring into paging/sort/filter/projection parameters. It
// returns an error for a filter that names an operator outside filterOperators, so a list
// handler can answer 400 instead of silently ignoring or mishandling the filter.
func GetQueryParams(query string) (QueryParams, error) {
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
		field, operation, err := parseFilterKey(k)
		if err != nil {
			return QueryParams{}, err
		}
		filters = append(filters, Filter{Field: field, Operation: operation, Value: v})
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
	return params, nil
}
