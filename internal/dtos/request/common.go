package request

type PaginationDto struct {
	Page   int    `json:"page,omitempty" query:"page" validate:"omitempty,min=1"`
	Limit  int    `json:"limit,omitempty" query:"limit" validate:"omitempty,min=1,max=100"`
	Offset int    `json:"offset,omitempty" query:"offset" validate:"omitempty,min=0"`
	Order  string `json:"order,omitempty" query:"order" validate:"omitempty,oneof=asc desc"`
}

func (p *PaginationDto) GetPage() int {
	if p.Page < 1 {
		return 1
	}
	return p.Page
}

func (p *PaginationDto) GetLimit(defaultLimit int) int {
	if p.Limit < 1 || p.Limit > 100 {
		if defaultLimit > 0 {
			return defaultLimit
		}
		return 20
	}
	return p.Limit
}

func (p *PaginationDto) GetOffset(defaultLimit int) int {
	if p.Offset > 0 {
		return p.Offset
	}
	return (p.GetPage() - 1) * p.GetLimit(defaultLimit)
}
