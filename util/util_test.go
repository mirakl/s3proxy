package util

import (
	"github.com/stretchr/testify/assert"
	"testing"
)

func TestArr2map(t *testing.T) {

	amap := Array2map("one", "two", "three")

	assert.Len(t, amap, 3)

	_, found := amap["one"]
	assert.True(t, found)

	_, found = amap["two"]
	assert.True(t, found)

	_, found = amap["three"]
	assert.True(t, found)

	_, found = amap["four"]
	assert.False(t, found)
}
