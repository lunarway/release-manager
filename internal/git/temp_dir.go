package git

import (
	"io/ioutil"
	"os"

	"github.com/lunarway/release-manager/internal/log"
)

// TempDir returns a temporary directory with provided prefix.
// The first return argument is the path. The second is a close function to
// remove the path.
func TempDir(prefix string) (string, func(), error) {
	path, err := ioutil.TempDir("", prefix)
	if err != nil {
		return "", func() {}, err
	}
	return path, func() {
		err := os.RemoveAll(path)
		if err != nil {
			log.Errorf("Removing temporary directory failed: path '%s': %v", path, err)
		}
	}, nil
}
