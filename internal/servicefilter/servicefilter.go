package servicefilter

type ServiceFilter interface {
	IsIncluded(service string) bool
}
