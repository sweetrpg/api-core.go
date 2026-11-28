package util

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"github.com/sweetrpg/common.go/logging"
)

// Encoded delimiters are data: the query is split first and each piece decoded afterwards.
func TestGetQueryParamsDecodesAfterSplitting(t *testing.T) {
	logging.Init()

	cases := []struct {
		name, query string
		want        []string
	}{
		{"ampersand", "filter[title]=Dungeons%20%26%20Dragons", []string{"Dungeons & Dragons"}},
		{"ampersand before parameter text", "filter[title]=a%26filter%5Bnotes%5D%3Db%26sort%3Dc%26page%5Blimit%5D%3D1",
			[]string{"a&filter[notes]=b&sort=c&page[limit]=1"}},
		{"encoded comma", "filter[title]=Hello%2C%20World", []string{"Hello, World"}},
		{"literal comma is a list", "filter[title]=a,b%2Cc, d", []string{"a", "b,c", "d"}},
		{"equals and plus", "filter[title]=1%2B1%3D2+ok", []string{"1+1=2 ok"}},
		{"percent decoded once", "filter[title]=100%2541", []string{"100%41"}},
		{"encoded key", "filter%5Btitle%5D=x", []string{"x"}},
		{"full URL", "http://localhost:1234/endpoint?filter[title]=x", []string{"x"}},
		{"literal question mark", "filter[title]=why?now", []string{"why?now"}},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			params, err := GetQueryParams(tc.query)
			require.NoError(t, err)
			require.Len(t, params.Filter, 1)
			assert.Equal(t, "title", params.Filter[0].Field)
			assert.Equal(t, tc.want, params.Filter[0].Value)
			assert.Empty(t, params.Sort)
			assert.EqualValues(t, 50, params.Limit)
		})
	}
}

func TestGetQueryParamsDecodesSortAndFieldsAfterSplitting(t *testing.T) {
	logging.Init()

	params, err := GetQueryParams("sort=a,-b&sort=c&fields=d,e&page[start]=2&page%5Blimit%5D=3&other=x%26sort%3Dz")
	require.NoError(t, err)
	assert.Equal(t, []Sort{{"a", 1}, {"-b", 1}, {"c", 1}}, params.Sort)
	assert.Equal(t, []Projection{{"d", true}, {"e", true}}, params.Projection)
	assert.EqualValues(t, 2, params.Start)
	assert.EqualValues(t, 3, params.Limit)
}

// Dropping an unreadable parameter would widen the result, so it is an error.
func TestGetQueryParamsRejectsUnreadableParameters(t *testing.T) {
	logging.Init()

	for _, query := range []string{"filter[title]=%zz", "filter%zz=x", "page[limit]=abc"} {
		_, err := GetQueryParams(query)
		assert.Error(t, err, query)
	}
}
