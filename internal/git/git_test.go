package git

import (
	"testing"

	"github.com/pkg/errors"
	"github.com/stretchr/testify/assert"
)

func TestCredentials(t *testing.T) {
	tt := []struct {
		name     string
		paths    []string
		userName string
		email    string
		err      error
	}{
		{
			name: "complete set",
			paths: []string{
				"testdata/user_set_1",
			},
			userName: "Foo",
			email:    "foo@foo.com",
			err:      nil,
		},
		{
			name: "first path missing email",
			paths: []string{
				"testdata/email_missing",
				"testdata/user_set_1",
			},
			userName: "Foo",
			email:    "foo@foo.com",
			err:      nil,
		},
		{
			name: "first path missing name",
			paths: []string{
				"testdata/name_missing",
				"testdata/user_set_1",
			},
			userName: "Foo",
			email:    "foo@foo.com",
			err:      nil,
		},
		{
			name: "configuration file not found in first path",
			paths: []string{
				"testdata/unknown_path",
				"testdata/user_set_1",
			},
			userName: "Foo",
			email:    "foo@foo.com",
			err:      nil,
		},
		{
			name: "configuration file not found in all paths",
			paths: []string{
				"testdata/unknown_path_1",
				"testdata/unknown_path_2",
			},
			userName: "",
			email:    "",
			err:      errors.New("failed to read Git credentials from paths: [testdata/unknown_path_1 testdata/unknown_path_2]"),
		},
	}
	for _, tc := range tt {
		t.Run(tc.name, func(t *testing.T) {
			userName, email, err := credentials(tc.paths...)
			t.Logf("error: %v", err)
			if tc.err != nil {
				assert.EqualError(t, err, tc.err.Error(), "output error not as expected")
			} else {
				assert.NoError(t, err, "unexpected output error")
			}
			assert.Equal(t, tc.userName, userName, "user name not as expected")
			assert.Equal(t, tc.email, email, "email not as expected")
		})
	}
}
