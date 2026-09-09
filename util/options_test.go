package util

import (
	"strings"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/suite"

	"github.com/sweetrpg/common.go/logging"
	dbconstants "github.com/sweetrpg/mongodb.go/constants"
)

type OptionsTestSuite struct {
	suite.Suite
}

func (suite *OptionsTestSuite) TestValidParams() {
	logging.Init()

	query := "http://localhost:1234/endpoint?page[start]=1&page[limit]=5"

	params, err := GetQueryParams(query)
	assert.NoError(suite.T(), err)
	assert.EqualValues(suite.T(), 1, params.Start)
	assert.EqualValues(suite.T(), 5, params.Limit)
}

func (suite *OptionsTestSuite) TestNoParams() {
	logging.Init()

	query := "http://localhost:1234/endpoint"

	params, err := GetQueryParams(query)
	assert.NoError(suite.T(), err)
	assert.EqualValues(suite.T(), 0, params.Start)
	assert.EqualValues(suite.T(), dbconstants.QueryDefaultSize, params.Limit)
}

func (suite *OptionsTestSuite) TestLowStart() {
	logging.Init()

	query := "http://localhost:1234/endpoint?page[start]=-5&page[limit]=5"

	params, err := GetQueryParams(query)
	assert.NoError(suite.T(), err)
	assert.EqualValues(suite.T(), 0, params.Start)
	assert.EqualValues(suite.T(), 5, params.Limit)
}

func (suite *OptionsTestSuite) TestOnlyStart() {
	logging.Init()

	query := "http://localhost:1234/endpoint?page[start]=1"

	params, err := GetQueryParams(query)
	assert.NoError(suite.T(), err)
	assert.EqualValues(suite.T(), 1, params.Start)
	assert.EqualValues(suite.T(), dbconstants.QueryDefaultSize, params.Limit)
}

func (suite *OptionsTestSuite) TestLowLimit() {
	logging.Init()

	query := "http://localhost:1234/endpoint?page[start]=1&page[limit]=-1"

	params, err := GetQueryParams(query)
	assert.NoError(suite.T(), err)
	assert.EqualValues(suite.T(), 1, params.Start)
	assert.EqualValues(suite.T(), 1, params.Limit)
}

func (suite *OptionsTestSuite) TestZeroLimit() {
	logging.Init()

	query := "http://localhost:1234/endpoint?page[start]=1&page[limit]=0"

	params, err := GetQueryParams(query)
	assert.NoError(suite.T(), err)
	assert.EqualValues(suite.T(), 1, params.Start)
	assert.EqualValues(suite.T(), dbconstants.QueryDefaultSize, params.Limit)
}

func (suite *OptionsTestSuite) TestHighLimit() {
	logging.Init()

	query := "http://localhost:1234/endpoint?page[start]=1&page[limit]=500"

	params, err := GetQueryParams(query)
	assert.NoError(suite.T(), err)
	assert.EqualValues(suite.T(), 1, params.Start)
	assert.EqualValues(suite.T(), dbconstants.QueryMaxSize, params.Limit)
}

func (suite *OptionsTestSuite) TestOnlyLImit() {
	logging.Init()

	query := "http://localhost:1234/endpoint?page[limit]=5"

	params, err := GetQueryParams(query)
	assert.NoError(suite.T(), err)
	assert.EqualValues(suite.T(), 0, params.Start)
	assert.EqualValues(suite.T(), 5, params.Limit)
}

// Task 1.1: the bare filter[field]=value form keeps parsing to Operation == nil (default $eq).
func (suite *OptionsTestSuite) TestBareFilterFormHasNilOperation() {
	logging.Init()

	params, err := GetQueryParams("http://localhost:1234/endpoint?filter[title]=Dune")
	assert.NoError(suite.T(), err)
	assert.Equal(suite.T(), 1, len(params.Filter))
	assert.Equal(suite.T(), "title", params.Filter[0].Field)
	assert.Nil(suite.T(), params.Filter[0].Operation)
	assert.EqualValues(suite.T(), []string{"Dune"}, params.Filter[0].Value)
}

// Task 1.1: the explicit filter[field][eq]=value form maps to $eq via the allow-list.
func (suite *OptionsTestSuite) TestExplicitEqOperatorMapsToMongoEq() {
	logging.Init()

	params, err := GetQueryParams("http://localhost:1234/endpoint?filter[title][eq]=Dune")
	assert.NoError(suite.T(), err)
	assert.Equal(suite.T(), 1, len(params.Filter))
	assert.Equal(suite.T(), "title", params.Filter[0].Field)
	if assert.NotNil(suite.T(), params.Filter[0].Operation) {
		assert.Equal(suite.T(), "$eq", *params.Filter[0].Operation)
	}
}

// Task 1.2: an operator segment outside the allow-list is an error, not a pass-through.
func (suite *OptionsTestSuite) TestUnrecognizedOperatorSegmentErrors() {
	logging.Init()

	_, err := GetQueryParams("http://localhost:1234/endpoint?filter[title][where]=x")
	assert.Error(suite.T(), err)
}

// Task 1.2: the client segment never becomes the Mongo operator - contains maps to $regex.
func (suite *OptionsTestSuite) TestContainsOperatorMapsToRegex() {
	logging.Init()

	params, err := GetQueryParams("http://localhost:1234/endpoint?filter[title][contains]=foo")
	assert.NoError(suite.T(), err)
	if assert.Equal(suite.T(), 1, len(params.Filter)) && assert.NotNil(suite.T(), params.Filter[0].Operation) {
		assert.Equal(suite.T(), "title", params.Filter[0].Field)
		assert.Equal(suite.T(), "$regex", *params.Filter[0].Operation)
	}
}

// A "$"-prefixed filter field is rejected, so a client can't reach $expr/$where/$or in
// operator position even though the operator segment itself is allow-listed.
func (suite *OptionsTestSuite) TestDollarPrefixedFilterFieldErrors() {
	logging.Init()

	_, err := GetQueryParams("http://localhost:1234/endpoint?filter[$expr][eq]=x")
	assert.Error(suite.T(), err)
}

// An empty path segment (leading/trailing/doubled dot) is rejected; a legitimate nested path
// like tags.value is not.
func (suite *OptionsTestSuite) TestFilterFieldSegmentValidation() {
	logging.Init()

	_, err := GetQueryParams("http://localhost:1234/endpoint?filter[tags.][eq]=x")
	assert.Error(suite.T(), err)

	params, err := GetQueryParams("http://localhost:1234/endpoint?filter[tags.value][in]=core")
	assert.NoError(suite.T(), err)
	assert.Equal(suite.T(), "tags.value", params.Filter[0].Field)
}

// A "$"-prefixed sort field is rejected.
func (suite *OptionsTestSuite) TestDollarPrefixedSortFieldErrors() {
	logging.Init()

	_, err := GetQueryParams("http://localhost:1234/endpoint?sort=$where")
	assert.Error(suite.T(), err)

	// the descending-sort "-" prefix is still fine
	params, err := GetQueryParams("http://localhost:1234/endpoint?sort=-name")
	assert.NoError(suite.T(), err)
	assert.Equal(suite.T(), "-name", params.Sort[0].Field)
}

// A filter value past the length cap is a 400, not a query.
func (suite *OptionsTestSuite) TestOverlongFilterValueErrors() {
	logging.Init()

	long := strings.Repeat("a", maxFilterValueLen+1)
	_, err := GetQueryParams("http://localhost:1234/endpoint?filter[title][contains]=" + long)
	assert.Error(suite.T(), err)
}

func TestOptionsTestSuite(t *testing.T) {
	suite.Run(t, new(OptionsTestSuite))
}
