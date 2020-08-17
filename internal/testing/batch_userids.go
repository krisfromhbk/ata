package testing

// BatchUserIDs splits single userIDs slice into several slices of two userIDs where first one is the first provided
// userID e.g. [0, 1, 2, 3, 4, 5] -> [[0,1], [0,2], [0,3], [0,4], [0,5]]
func BatchUserIDs(userIDs []int64) [][]int64 {
	batches := make([][]int64, 0, len(userIDs)-1)
	for i := 1; i < len(userIDs); i++ {
		batches = append(batches, []int64{userIDs[0], userIDs[i]})
	}

	return batches
}
