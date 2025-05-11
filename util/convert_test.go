package util

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/suite"
	"github.com/sweetrpg/common.go/logging"
)

type ConvertTestSuite struct {
	suite.Suite
}

func (suite *ConvertTestSuite) TestConvertNoParams() {
	logging.Init()

	query := "http://localhost:1234/endpoint"

	params := GetQueryParams(query)
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

	params := GetQueryParams(query)
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

	params := GetQueryParams(query)
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

	params := GetQueryParams(query)
	logging.Logger.Debug("GetQueryParams", "params", params)
	assert.EqualValues(suite.T(), 0, params.Start)
	assert.EqualValues(suite.T(), 50, params.Limit)

	filter, sort, projection := ConvertQueryParams(params)
	logging.Logger.Debug("ConvertQueryParams", "filter", filter, "sort", sort, "projection", projection)
	assert.Empty(suite.T(), sort)
	assert.Empty(suite.T(), projection)

	assert.Equal(suite.T(), 1, len(filter))
	assert.EqualValues(suite.T(), "baz", filter[0].Key)
	assert.EqualValues(suite.T(), []string{"1"}, filter[0].Value)
}

func (suite *ConvertTestSuite) TestConvertProjectionParams() {
	logging.Init()

	query := "http://localhost:1234/endpoint?fields=foo,bar,baz"

	params := GetQueryParams(query)
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

	params := GetQueryParams(query)
	logging.Logger.Debug("GetQueryParams", "params", params)
	assert.EqualValues(suite.T(), 1, params.Start)
	assert.EqualValues(suite.T(), 5, params.Limit)

	filter, sort, projection := ConvertQueryParams(params)
	logging.Logger.Debug("ConvertQueryParams", "filter", filter, "sort", sort, "projection", projection)

	assert.Equal(suite.T(), 1, len(filter))
	assert.EqualValues(suite.T(), "baz", filter[0].Key)
	assert.EqualValues(suite.T(), []string{"1"}, filter[0].Value)

	assert.Equal(suite.T(), 1, len(sort))
	assert.EqualValues(suite.T(), "bar", sort[0].Key)
	assert.EqualValues(suite.T(), 1, sort[0].Value)

	assert.Equal(suite.T(), 2, len(projection))
	assert.EqualValues(suite.T(), "foo", projection[0].Key)
	assert.EqualValues(suite.T(), "bar", projection[1].Key)
	assert.EqualValues(suite.T(), 1, projection[0].Value)
}

func TestConvertTestSuite(t *testing.T) {
	suite.Run(t, new(ConvertTestSuite))
}
