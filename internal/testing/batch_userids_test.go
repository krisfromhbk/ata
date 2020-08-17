package testing

import (
	"github.com/stretchr/testify/require"
	"testing"
)

func TestBatchUserIDs(t *testing.T) {
	userIDs := []int64{0, 1, 2, 3, 4, 5}
	batches := BatchUserIDs(userIDs)
	require.Equal(t, [][]int64{{0, 1}, {0, 2}, {0, 3}, {0, 4}, {0, 5}}, batches)
}
