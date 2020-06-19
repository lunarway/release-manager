package commitinfo

import (
	"github.com/pkg/errors"

	"github.com/lunarway/release-manager/internal/regexp"
)

type PersonInfo struct {
	Name  string
	Email string
}

func ParsePerson(personInfo string) (PersonInfo, error) {
	matches := personInfoRegex.FindStringSubmatch(personInfo)
	if matches == nil {
		return PersonInfo{}, errors.New("no match")
	}
	return PersonInfo{
		Name:  matches[personInfoRegexLookup.Name],
		Email: matches[personInfoRegexLookup.Email],
	}, nil
}

var personInfoRegexLookup = struct {
	Name  int
	Email int
}{}
var personInfoRegex = regexp.MustCompile(`^(?P<Name>.*)\s<(?P<Email>.*)>$`, &personInfoRegexLookup)
