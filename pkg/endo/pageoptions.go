package endo

// PageOptions represents a page and a per page limit.
type PageOptions struct {
	Page, PerPage int
}

// Args returns the limit and offset arguments for a query.
func (po *PageOptions) Args() (limit, offset int) {
	page := po.Page
	if page < 1 {
		page = 1
	}
	limit = po.PerPage
	if limit < 1 {
		limit = 10
	}
	offset = (page - 1) * limit
	return
}
