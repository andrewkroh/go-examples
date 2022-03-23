package ecs

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestGetFieldSet(t *testing.T) {
	fields := GetFieldSet("source")
	assert.NotEmpty(t, fields)
}
