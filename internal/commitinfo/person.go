package commitinfo

import (
	"fmt"

	"github.com/lunarway/release-manager/internal/regexp"
	"github.com/pkg/errors"
)

type PersonInfo struct {
	Name  string
	Email string
}

func NewPersonInfo(name, email string) PersonInfo {
	return PersonInfo{
		Name:  name,
		Email: email,
	}
}

func (i PersonInfo) String() string {
	return fmt.Sprintf("%s <%s>", i.Name, i.Email)
}

func ParsePerson(personInfo string) (PersonInfo, error) {
	matches := personInfoRegex.FindStringSubmatch(personInfo)
	if matches == nil {
		return PersonInfo{}, errors.Wrap(ErrNoMatch, fmt.Sprintf("string '%s' does not contain a person", personInfo))
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
var personInfoRegex = regexp.MustCompile(`^(?P<Name>.*) <(?P<Email>.*)>$`, &personInfoRegexLookup)
