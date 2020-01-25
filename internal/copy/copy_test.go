package copy_test

import (
	"os"
	"path"
	"testing"

	"github.com/lunarway/release-manager/internal/copy"
	"github.com/lunarway/release-manager/internal/log"
	"github.com/stretchr/testify/assert"
	"go.uber.org/zap/zapcore"
)

func TestCopyDir(t *testing.T) {
	tt := []struct {
		name string
		src  string
		dest string
		err  error
	}{
		{
			name: "existing directory",
			src:  "testdata/dir-a",
			dest: "testdata/new-dir-a",
			err:  nil,
		},
		{
			name: "unknown source directory",
			src:  "testdata/dir-unknown",
			dest: "testdata/new-dir-a",
			err:  copy.ErrUnknownSource,
		},
	}
	for _, tc := range tt {
		t.Run(tc.name, func(t *testing.T) {
			log.Init(&log.Configuration{
				Level: log.Level{
					Level: zapcore.DebugLevel,
				},
				Development: true,
			})
			pwd, err := os.Getwd()
			if err != nil {
				t.Fatalf("unexpeted error getting pwd: %v", err)
			}
			t.Logf("Using pwd: %s", pwd)
			absSrc := path.Join(pwd, tc.src)
			absDest := path.Join(pwd, tc.dest)
			err = copy.CopyDir(absSrc, absDest)
			defer os.RemoveAll(absDest)
			if tc.err != nil {
				assert.EqualError(t, err, tc.err.Error(), "wrong error")
			} else {
				assert.NoError(t, err, "unepxected error")
			}
		})
	}
}
