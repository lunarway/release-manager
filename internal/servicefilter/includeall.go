package servicefilter

type includeAll struct{}

func NewIncludeAll() ServiceFilter {
	return &includeAll{}
}

func (includeAll) IsIncluded(service string) bool {
	return true
}