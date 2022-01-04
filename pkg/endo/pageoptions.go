package endo

type PageOptions struct {
	Page, PerPage int
}

func (po *PageOptions) Args() (limit, offset int32) {
	page := po.Page
	if page < 1 {
		page = 1
	}
	limit = int32(po.PerPage)
	if 1 > limit || limit > 100 {
		limit = 100
	}
	offset = (int32(page) - 1) * limit
	return
}
