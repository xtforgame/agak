package utils

type StringSlice []string

func (ss StringSlice) Len() int {
	return len(ss)
}
func (ss StringSlice) Swap(i, j int) {
	ss[i], ss[j] = ss[j], ss[i]
}
func (ss StringSlice) Less(i, j int) bool {
	return ss[i] < ss[j]
}
