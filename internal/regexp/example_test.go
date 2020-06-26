package regexp_test

import (
	"fmt"

	"github.com/lunarway/release-manager/internal/regexp"
)

func ExampleCompile() {
	var lookup = struct {
		FirstPart  int
		SecondPart int
	}{}
	re, err := regexp.Compile(`(?P<FirstPart>.+)-(?P<SecondPart>.+)`, &lookup) // Handle err or use MustCompile
	if err != nil {
		panic(err)
	}
	matches := re.FindStringSubmatch("some-message")
	if matches == nil {
		panic("no match")
	}

	fmt.Println(matches[lookup.FirstPart], matches[lookup.SecondPart])
	// Output: some message
}
