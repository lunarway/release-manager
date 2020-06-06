package servicefilter

import (
	"regexp"
)

type regex struct {
	re *regexp.Regexp
}

func FromStringRegex(regexString string) (ServiceFilter, error) {
	re, err := regexp.Compile(`^\[(?P<service>.*)\]( artifact (?P<artifactID>[^ ]+) by)?.*\nArtifact-created-by:\s(?P<authorName>.*)\s<(?P<authorEmail>.*)>`)

	if err != nil {
		return nil, err
	}
	return &regex{
		re: re,
	}, nil
}

func (r *regex) IsIncluded(service string) bool {
	return false
}
