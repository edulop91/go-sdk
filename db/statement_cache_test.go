package db

import (
	"context"
	"testing"

	assert "github.com/blend/go-sdk/assert"
)

func TestStatementCachePrepare(t *testing.T) {
	assert := assert.New(t)

	sc := NewStatementCache().WithConnection(Default().Connection())

	query := "select 'ok'"
	stmt, err := sc.PrepareContext(context.Background(), query, query, nil)
	assert.Nil(err)
	assert.NotNil(stmt)
	assert.True(sc.HasStatement(query))

	// shoul result in cache hit
	stmt, err = sc.PrepareContext(context.Background(), query, query, nil)
	assert.NotNil(stmt)
	assert.True(sc.HasStatement(query))
}
