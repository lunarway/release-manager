package http

import (
	"fmt"
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestAuthenticate(t *testing.T) {
	tt := []struct {
		name          string
		serverToken   string
		authorization string
		status        error
	}{
		{
			name:          "empty authorization",
			serverToken:   "token",
			authorization: "",
			status:        fmt.Errorf("please provide a valid authentication token"),
		},
		{
			name:          "whitespace token",
			serverToken:   "token",
			authorization: "  ",
			status:        fmt.Errorf("please provide a valid authentication token"),
		},
		{
			name:          "non-bearer authorization",
			serverToken:   "token",
			authorization: "non-bearer-token",
			status:        fmt.Errorf("please provide a valid authentication token"),
		},
		{
			name:          "empty bearer authorization",
			serverToken:   "token",
			authorization: "Bearer ",
			status:        fmt.Errorf("please provide a valid authentication token"),
		},
		{
			name:          "whitespace bearer authorization",
			serverToken:   "token",
			authorization: "Bearer      ",
			status:        fmt.Errorf("please provide a valid authentication token"),
		},
		{
			name:          "wrong bearer authorization",
			serverToken:   "token",
			authorization: "Bearer another-token",
			status:        fmt.Errorf("please provide a valid authentication token"),
		},
		{
			name:          "correct bearer authorization",
			serverToken:   "token",
			authorization: "Bearer token",
			status:        nil,
		},
	}
	for _, tc := range tt {
		t.Run(tc.name, func(t *testing.T) {
			_, err := authenticate(tc.serverToken)(tc.authorization)

			if tc.status != nil {
				assert.EqualError(t, err, tc.status.Error())
			} else {
				assert.NoError(t, err)
			}
		})
	}
}
