package utils

import "math/rand"

func ShuffleList[T any](slice []T) {
	rand.Shuffle(len(slice), func(i, j int) {
		slice[i], slice[j] = slice[j], slice[i]
	})
}

func RemoveByIndex[T any](s []T, index int) []T {
	return append(s[:index], s[index+1:]...)
}
