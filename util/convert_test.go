package util

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/suite"
	"github.com/sweetrpg/common.go/logging"
	"go.mongodb.org/mongo-driver/bson"
)

type ConvertTestSuite struct {
	suite.Suite
}

func (suite *ConvertTestSuite) TestConvertNoParams() {
	logging.Init()

	query := "http://localhost:1234/endpoint"

	params, err := GetQueryParams(query)
	assert.NoError(suite.T(), err)
	logging.Logger.Debug("GetQueryParams", "params", params)
	assert.EqualValues(suite.T(), 0, params.Start)
	assert.EqualValues(suite.T(), 50, params.Limit)

	filter, sort, projection := ConvertQueryParams(params)
	logging.Logger.Debug("ConvertQueryParams", "filter", filter, "sort", sort, "projection", projection)
	assert.Empty(suite.T(), filter)
	assert.Empty(suite.T(), sort)
	assert.Empty(suite.T(), projection)
}

func (suite *ConvertTestSuite) TestConvertPagingParams() {
	logging.Init()

	query := "http://localhost:1234/endpoint?page[start]=1&page[limit]=5"

	params, err := GetQueryParams(query)
	assert.NoError(suite.T(), err)
	logging.Logger.Debug("GetQueryParams", "params", params)
	assert.EqualValues(suite.T(), 1, params.Start)
	assert.EqualValues(suite.T(), 5, params.Limit)

	filter, sort, projection := ConvertQueryParams(params)
	logging.Logger.Debug("ConvertQueryParams", "filter", filter, "sort", sort, "projection", projection)
	assert.Empty(suite.T(), filter)
	assert.Empty(suite.T(), sort)
	assert.Empty(suite.T(), projection)
}

func (suite *ConvertTestSuite) TestConvertSortParams() {
	logging.Init()

	query := "http://localhost:1234/endpoint?sort=bar"

	params, err := GetQueryParams(query)
	assert.NoError(suite.T(), err)
	logging.Logger.Debug("GetQueryParams", "params", params)
	assert.EqualValues(suite.T(), 0, params.Start)
	assert.EqualValues(suite.T(), 50, params.Limit)

	filter, sort, projection := ConvertQueryParams(params)
	logging.Logger.Debug("ConvertQueryParams", "filter", filter, "sort", sort, "projection", projection)
	assert.Empty(suite.T(), filter)
	assert.Empty(suite.T(), projection)

	assert.Equal(suite.T(), 1, len(sort))
	assert.EqualValues(suite.T(), "bar", sort[0].Key)
	assert.EqualValues(suite.T(), 1, sort[0].Value)
}

func (suite *ConvertTestSuite) TestConvertFilterParams() {
	logging.Init()

	query := "http://localhost:1234/endpoint?filter[baz]=1"

	params, err := GetQueryParams(query)
	assert.NoError(suite.T(), err)
	logging.Logger.Debug("GetQueryParams", "params", params)
	assert.EqualValues(suite.T(), 0, params.Start)
	assert.EqualValues(suite.T(), 50, params.Limit)

	filter, sort, projection := ConvertQueryParams(params)
	logging.Logger.Debug("ConvertQueryParams", "filter", filter, "sort", sort, "projection", projection)
	assert.Empty(suite.T(), sort)
	assert.Empty(suite.T(), projection)

	assert.Equal(suite.T(), 1, len(filter))
	assert.EqualValues(suite.T(), "baz", filter[0].Key)
	assert.EqualValues(suite.T(), bson.D{{Key: "$eq", Value: []string{"1"}}}, filter[0].Value)
}

func (suite *ConvertTestSuite) TestConvertProjectionParams() {
	logging.Init()

	query := "http://localhost:1234/endpoint?fields=foo,bar,baz"

	params, err := GetQueryParams(query)
	assert.NoError(suite.T(), err)
	logging.Logger.Debug("GetQueryParams", "params", params)
	assert.EqualValues(suite.T(), 0, params.Start)
	assert.EqualValues(suite.T(), 50, params.Limit)

	filter, sort, projection := ConvertQueryParams(params)
	logging.Logger.Debug("ConvertQueryParams", "filter", filter, "sort", sort, "projection", projection)
	assert.Empty(suite.T(), filter)
	assert.Empty(suite.T(), sort)

	assert.Equal(suite.T(), 3, len(projection))
	assert.EqualValues(suite.T(), "foo", projection[0].Key)
	assert.EqualValues(suite.T(), "bar", projection[1].Key)
	assert.EqualValues(suite.T(), "baz", projection[2].Key)
	assert.EqualValues(suite.T(), 1, projection[0].Value)
}

func (suite *ConvertTestSuite) TestConvertAllParams() {
	logging.Init()

	query := "http://localhost:1234/endpoint?page[start]=1&page[limit]=5&fields=foo,bar&sort=bar&filter[baz]=1"

	params, err := GetQueryParams(query)
	assert.NoError(suite.T(), err)
	logging.Logger.Debug("GetQueryParams", "params", params)
	assert.EqualValues(suite.T(), 1, params.Start)
	assert.EqualValues(suite.T(), 5, params.Limit)

	filter, sort, projection := ConvertQueryParams(params)
	logging.Logger.Debug("ConvertQueryParams", "filter", filter, "sort", sort, "projection", projection)

	assert.Equal(suite.T(), 1, len(filter))
	assert.EqualValues(suite.T(), "baz", filter[0].Key)
	assert.EqualValues(suite.T(), bson.D{{Key: "$eq", Value: []string{"1"}}}, filter[0].Value)

	assert.Equal(suite.T(), 1, len(sort))
	assert.EqualValues(suite.T(), "bar", sort[0].Key)
	assert.EqualValues(suite.T(), 1, sort[0].Value)

	assert.Equal(suite.T(), 2, len(projection))
	assert.EqualValues(suite.T(), "foo", projection[0].Key)
	assert.EqualValues(suite.T(), "bar", projection[1].Key)
	assert.EqualValues(suite.T(), 1, projection[0].Value)
}

func (suite *ConvertTestSuite) TestConvertFilterOperatorMarshalsAsMongoOperator() {
	op := "$gt"
	params := QueryParams{
		Filter: []Filter{{Field: "age", Operation: &op, Value: []string{"21"}}},
	}

	filter, _, _ := ConvertQueryParams(params)

	// Regression check: the filter value must marshal as a Mongo operator document
	// ({"$gt": [...]}), not as a struct with literal "key"/"value" fields.
	doc, err := bson.MarshalExtJSON(filter, false, false)
	assert.NoError(suite.T(), err)
	assert.JSONEq(suite.T(), `{"age":{"$gt":["21"]}}`, string(doc))
}

func (suite *ConvertTestSuite) TestConvertProjectionExclusionMarshalsAsZero() {
	params := QueryParams{
		Projection: []Projection{{Field: "secret", Inclusion: false}},
	}

	_, _, projection := ConvertQueryParams(params)

	assert.Equal(suite.T(), 1, len(projection))
	assert.EqualValues(suite.T(), "secret", projection[0].Key)
	assert.EqualValues(suite.T(), 0, projection[0].Value)
}

// Task 1.3: filter[title][contains]=foo converts to a case-insensitive $regex clause.
func (suite *ConvertTestSuite) TestConvertContainsBuildsCaseInsensitiveRegex() {
	logging.Init()

	params, err := GetQueryParams("http://localhost:1234/endpoint?filter[title][contains]=foo")
	assert.NoError(suite.T(), err)

	filter, _, _ := ConvertQueryParams(params)

	assert.Equal(suite.T(), 1, len(filter))
	assert.EqualValues(suite.T(), "title", filter[0].Key)
	assert.EqualValues(suite.T(), bson.D{{Key: "$regex", Value: "foo"}, {Key: "$options", Value: "i"}}, filter[0].Value)

	doc, marshalErr := bson.MarshalExtJSON(filter, false, false)
	assert.NoError(suite.T(), marshalErr)
	assert.JSONEq(suite.T(), `{"title":{"$regex":"foo","$options":"i"}}`, string(doc))
}

// Task 1.1 regression at the convert layer: the bare form still yields $eq.
func (suite *ConvertTestSuite) TestConvertBareFilterStillEq() {
	logging.Init()

	params, err := GetQueryParams("http://localhost:1234/endpoint?filter[title]=Dune")
	assert.NoError(suite.T(), err)

	filter, _, _ := ConvertQueryParams(params)

	assert.Equal(suite.T(), 1, len(filter))
	assert.EqualValues(suite.T(), "title", filter[0].Key)
	assert.EqualValues(suite.T(), bson.D{{Key: "$eq", Value: []string{"Dune"}}}, filter[0].Value)
}

func TestConvertTestSuite(t *testing.T) {
	suite.Run(t, new(ConvertTestSuite))
}
