package regexp_test

import (
	"fmt"
	"testing"

	"github.com/lunarway/release-manager/internal/regexp"
	"github.com/pkg/errors"
	"github.com/stretchr/testify/assert"
)

type lookupCallback func(lookup interface{}, test func(a *assert.Assertions))

func TestCompile(t *testing.T) {
	testCases := []struct {
		desc   string
		regexp string
		lookup func(lookupCallback lookupCallback)
		err    error
	}{
		{
			desc:   "using no pointer to lookup",
			regexp: `(?P<Test>)`,
			lookup: func(cb lookupCallback) {
				l := struct{ Test int }{}
				cb(l, nil)
			},
			err: fmt.Errorf("lookup must be a pointer"),
		},
		{
			desc:   "using private or lower case property gives error",
			regexp: `(?P<test>)`,
			lookup: func(cb lookupCallback) {
				l := struct{ test int }{}
				cb(&l, nil)
			},
			err: fmt.Errorf("field 'test' in regexp `(?P<test>)` is not capitalized"),
		},
		{
			desc:   "gives error when regexp has named groups lookup don't have",
			regexp: `(?P<Other>1337)(?P<Test>.*)`,
			lookup: func(cb lookupCallback) {
				l := struct {
					Test int
				}{}
				cb(&l, nil)
			},
			err: fmt.Errorf("regexp `(?P<Other>1337)(?P<Test>.*)` has named groups not found in lookup: Other"),
		},
		{
			desc:   "gives error when lookup has properties regexp don't have",
			regexp: `(?P<Test>.*)`,
			lookup: func(cb lookupCallback) {
				l := struct {
					Test  int
					Other int
				}{}
				cb(&l, nil)
			},
			err: fmt.Errorf("field 'Other' is not in regexp `(?P<Test>.*)`"),
		},
		{
			desc:   "lookup int properties should match",
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
			_, err := regexp.Compile(tC.regexp, lookup)
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
