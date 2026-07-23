package server

import (
	"os"
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestPingHandler(t *testing.T) {
	vo := PingHandler()

	assert.NotNil(t, vo)
	assert.NotNil(t, vo.Date)

	hostname, _ := os.Hostname()
	assert.Equal(t, hostname, vo.Hostname)
}
