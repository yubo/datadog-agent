package util

import "sort"

const insertionSortThreshold = 20

// SortUniqInPlace sorts and remove duplicates from elements in place
// The returned slice is a subslice of elements
func SortUniqInPlace(elements *StringSlice) {
	size := elements.Len()
	if size < 2 {
		return
	}
	if size <= insertionSortThreshold {
		insertionSort(elements)
	} else {
		// this will trigger an alloc because sorts uses interface{} internaly
		// which confuses the escape analysis
		sort.Strings(elements.Slice())
	}
	uniqSorted(elements)
}

func insertionSort(elements *StringSlice) {
	for i := uint(1); i < elements.Len(); i++ {
		temp := elements.Get(i)
		j := uint(i)
		for j > 0 && temp <= elements.Get(j-1) {
			elements.Set(j, elements.Get(j-1))
			j--
		}
		elements.Set(j, temp)
	}
}

// uniqSorted remove duplicate elements from the given slice
// the given slice needs to be sorted
func uniqSorted(elements *StringSlice) {
	j := uint(0)
	for i := uint(1); i < elements.Len(); i++ {
		if elements.Get(j) == elements.Get(i) {
			continue
		}
		j++
		elements.Set(j, elements.Get(i))
	}
	elements.Resize(j + 1)
}
