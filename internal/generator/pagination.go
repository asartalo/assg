package generator

func PaginateTransform[S ~[]E, E any, F any](collection S, perPage int, transformer func(input E) F) [][]F {
	if perPage <= 0 {
		return [][]F{}
	}

	totalItems := len(collection)
	totalPages := (totalItems + perPage - 1) / perPage
	pages := make([][]F, 0, totalPages)

	page := make([]F, 0, perPage)
	for i, item := range collection {
		page = append(page, transformer(item))

		if len(page) == perPage || i == totalItems-1 {
			pages = append(pages, page)
			page = make([]F, 0, perPage)
		}
	}

	return pages
}
