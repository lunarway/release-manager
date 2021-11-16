package flow

import (
	"io/fs"
	"os"
	"path/filepath"
	"strings"

	yaml "gopkg.in/yaml.v2"
)

type kustomizationSpec struct {
	APIVersion string `yaml:"apiVersion"`
	Kind       string `yaml:"kind"`
}

func kustomizationExists(directory string) (string, error) {
	var filePath string

	err := filepath.WalkDir(directory, func(path string, d fs.DirEntry, err error) error {
		if err != nil {
			return err
		}

		if filePath != "" {
			return nil
		}

		if d.IsDir() {
			return nil
		}

		file, err := os.OpenFile(path, os.O_RDWR, os.ModePerm)
		if err != nil {
			return err
		}

		var spec kustomizationSpec

		// no need to handle non-YAML files as the decoder does the right thing
		err = yaml.NewDecoder(file).Decode(&spec)
		if err != nil {
			return err
		}
		if strings.HasPrefix(spec.APIVersion, "kustomize.toolkit.fluxcd.io/") && spec.Kind == "Kustomization" {
			filePath = path
		}

		return nil
	})

	if err != nil {
		return "", err
	}

	return filePath, nil
}
