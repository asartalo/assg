package generator

import (
	"strconv"
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestPaginateTransform(t *testing.T) {
	arr := []int{1, 2, 3, 4, 5, 6, 7, 8}
	byPage := 3
	pages := PaginateTransform(arr, byPage, strconv.Itoa)

	assert.Equal(
		t, [][]string{{"1", "2", "3"}, {"4", "5", "6"}, {"7", "8"}}, pages,
	)
}

func TestPaginateTransformSinglePage(t *testing.T) {
	arr := []int{1, 2}
	byPage := 3
	pages := PaginateTransform(arr, byPage, strconv.Itoa)

	assert.Equal(t, [][]string{{"1", "2"}}, pages)
}

func TestPaginateSinglePageExact(t *testing.T) {
	arr := []int{1, 2}
	byPage := 2

	pages := PaginateTransform(arr, byPage, strconv.Itoa)
	assert.Equal(t, [][]string{{"1", "2"}}, pages)
}

func TestPaginateTranformExact2(t *testing.T) {
	arr := []int{1, 2, 3, 4}
	byPage := 2
	pages := PaginateTransform(arr, byPage, strconv.Itoa)

	assert.Equal(t, [][]string{{"1", "2"}, {"3", "4"}}, pages)
}
