package fuzzy

type Match struct {
	Index int
	Str   string
	Score int
	Pos   []int
}

func Find(pattern []rune, array []string) *Match {
	var result Result
	var pos *[]int
	foundIndex := -1
	for i := range array {
		input := ToChars([]byte(array[i]))
		r, p := fuzzyMatch(&input, pattern)
		if r.Score > result.Score {
			result = r
			pos = p
			foundIndex = i
		}
	}
	if foundIndex >= 0 && pos != nil {
		return &Match{
			Index: foundIndex,
			Str:   array[foundIndex],
			Score: result.Score,
			Pos:   *pos,
		}
	}
	return nil
}
