package regexp

import (
	"fmt"
	"testing"

	"github.com/pkg/errors"
	"github.com/stretchr/testify/assert"
)

type lookupCallback = func(lookup interface{}, test func(a *assert.Assertions))

func TestCompile(t *testing.T) {
	testCases := []struct {
		desc   string
		regexp string
		lookup func(lookupCallback lookupCallback)
		err    error
	}{
		{
			desc:   "using no pointer to lookup",
			regexp: `(?P<test>)`,
			lookup: func(cb lookupCallback) {
				l := struct{ Test int }{}
				cb(l, nil)
			},
			err: fmt.Errorf("lookup must be a pointer"),
		},
		{
			desc:   "simple",
			regexp: `(?P<Other>1337)(?P<Test>.*)`,
			lookup: func(cb lookupCallback) {
				l := struct {
					Test  int
					Other int
				}{}
				cb(&l, func(a *assert.Assertions) {
					a.Equal(l.Other, 1)
					a.Equal(l.Test, 2)
				})
			},
			err: nil,
		},
	}
	for _, tC := range testCases {
		t.Run(tC.desc, func(t *testing.T) {
			a := assert.New(t)
			var lookup interface{}
			var assertCallback func(t *assert.Assertions)
			tC.lookup(func(l interface{}, acb func(t *assert.Assertions)) {
				lookup = l
				assertCallback = acb
			})
			_, err := Compile(tC.regexp, lookup)
			if tC.err != nil {
				assert.EqualError(t, errors.Cause(err), tC.err.Error(), "output error not as expected")
			} else {
				assert.NoError(t, err, "no output error expected")
			}
			if assertCallback != nil {
				assertCallback(a)
			}
		})
	}
}
