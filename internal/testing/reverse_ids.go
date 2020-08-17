package testing

// ReverseIDs reverses provided ids
func ReverseIDs(ids []int64) []int64 {
	reversed := make([]int64, len(ids))
	copy(reversed, ids)

	for i := len(reversed)/2 - 1; i >= 0; i-- {
		opp := len(reversed) - 1 - i
		reversed[i], reversed[opp] = reversed[opp], reversed[i]
	}

	return reversed
}
