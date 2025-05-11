package util

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/suite"

	"github.com/sweetrpg/common.go/logging"
	dbconstants "github.com/sweetrpg/db.go/constants"
)

type OptionsTestSuite struct {
	suite.Suite
}

func (suite *OptionsTestSuite) TestValidParams() {
	logging.Init()

	query := "http://localhost:1234/endpoint?page[start]=1&page[limit]=5"

	params := GetQueryParams(query)
	assert.EqualValues(suite.T(), 1, params.Start)
	assert.EqualValues(suite.T(), 5, params.Limit)
}

func (suite *OptionsTestSuite) TestNoParams() {
	logging.Init()

	query := "http://localhost:1234/endpoint"

	params := GetQueryParams(query)
	assert.EqualValues(suite.T(), 0, params.Start)
	assert.EqualValues(suite.T(), dbconstants.QueryDefaultSize, params.Limit)
}

func (suite *OptionsTestSuite) TestLowStart() {
	logging.Init()

	query := "http://localhost:1234/endpoint?page[start]=-5&page[limit]=5"

	params := GetQueryParams(query)
	assert.EqualValues(suite.T(), 0, params.Start)
	assert.EqualValues(suite.T(), 5, params.Limit)
}

func (suite *OptionsTestSuite) TestOnlyStart() {
	logging.Init()

	query := "http://localhost:1234/endpoint?page[start]=1"

	params := GetQueryParams(query)
	assert.EqualValues(suite.T(), 1, params.Start)
	assert.EqualValues(suite.T(), dbconstants.QueryDefaultSize, params.Limit)
}

func (suite *OptionsTestSuite) TestLowLimit() {
	logging.Init()

	query := "http://localhost:1234/endpoint?page[start]=1&page[limit]=-1"

	params := GetQueryParams(query)
	assert.EqualValues(suite.T(), 1, params.Start)
	assert.EqualValues(suite.T(), 1, params.Limit)
}

func (suite *OptionsTestSuite) TestZeroLimit() {
	logging.Init()

	query := "http://localhost:1234/endpoint?page[start]=1&page[limit]=0"

	params := GetQueryParams(query)
	assert.EqualValues(suite.T(), 1, params.Start)
	assert.EqualValues(suite.T(), dbconstants.QueryDefaultSize, params.Limit)
}

func (suite *OptionsTestSuite) TestHighLimit() {
	logging.Init()

	query := "http://localhost:1234/endpoint?page[start]=1&page[limit]=500"

	params := GetQueryParams(query)
	assert.EqualValues(suite.T(), 1, params.Start)
	assert.EqualValues(suite.T(), dbconstants.QueryMaxSize, params.Limit)
}

func (suite *OptionsTestSuite) TestOnlyLImit() {
	logging.Init()

	query := "http://localhost:1234/endpoint?page[limit]=5"

	params := GetQueryParams(query)
	assert.EqualValues(suite.T(), 0, params.Start)
	assert.EqualValues(suite.T(), 5, params.Limit)
}

func TestOptionsTestSuite(t *testing.T) {
	suite.Run(t, new(OptionsTestSuite))
}
