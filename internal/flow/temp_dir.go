package flow

import (
	"io/ioutil"
	"os"

	"github.com/lunarway/release-manager/internal/log"
)

func tempDir(prefix string) (string, func(), error) {
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
