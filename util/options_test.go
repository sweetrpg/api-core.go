package util

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/sweetrpg/common.go/logging"
	dbconstants "github.com/sweetrpg/db.go/constants"
)

func TestValidParams(t *testing.T) {
	logging.Init()

	query := "http://localhost:1234/endpoint?page[start]=1&page[limit]=5"

	params := GetQueryParams(query)
	assert.EqualValues(t, 1, params.Start)
	assert.EqualValues(t, 5, params.Limit)
}

func TestNoParams(t *testing.T) {
	logging.Init()

	query := "http://localhost:1234/endpoint"

	params := GetQueryParams(query)
	assert.EqualValues(t, 0, params.Start)
	assert.EqualValues(t, dbconstants.QueryDefaultSize, params.Limit)
}

func TestLowStart(t *testing.T) {
	logging.Init()

	query := "http://localhost:1234/endpoint?page[start]=-5&page[limit]=5"

	params := GetQueryParams(query)
	assert.EqualValues(t, 0, params.Start)
	assert.EqualValues(t, 5, params.Limit)
}

func TestOnlyStart(t *testing.T) {
	logging.Init()

	query := "http://localhost:1234/endpoint?page[start]=1"

	params := GetQueryParams(query)
	assert.EqualValues(t, 1, params.Start)
	assert.EqualValues(t, dbconstants.QueryDefaultSize, params.Limit)
}

func TestLowLimit(t *testing.T) {
	logging.Init()

	query := "http://localhost:1234/endpoint?page[start]=1&page[limit]=-1"

	params := GetQueryParams(query)
	assert.EqualValues(t, 1, params.Start)
	assert.EqualValues(t, 1, params.Limit)
}

func TestZeroLimit(t *testing.T) {
	logging.Init()

	query := "http://localhost:1234/endpoint?page[start]=1&page[limit]=0"

	params := GetQueryParams(query)
	assert.EqualValues(t, 1, params.Start)
	assert.EqualValues(t, dbconstants.QueryDefaultSize, params.Limit)
}

func TestHighLimit(t *testing.T) {
	logging.Init()

	query := "http://localhost:1234/endpoint?page[start]=1&page[limit]=500"

	params := GetQueryParams(query)
	assert.EqualValues(t, 1, params.Start)
	assert.EqualValues(t, dbconstants.QueryMaxSize, params.Limit)
}

func TestOnlyLImit(t *testing.T) {
	logging.Init()

	query := "http://localhost:1234/endpoint?page[limit]=5"

	params := GetQueryParams(query)
	assert.EqualValues(t, 0, params.Start)
	assert.EqualValues(t, 5, params.Limit)
}
