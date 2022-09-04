package flow

import (
	"context"
	"fmt"
	"io"
	"io/fs"
	"os"
	"path/filepath"
	"strings"

	securejoin "github.com/cyphar/filepath-securejoin"
	"github.com/lunarway/release-manager/internal/log"
	"github.com/pkg/errors"
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
			return errors.WithMessagef(err, "walk dir '%s'", directory)
		}

		if filePath != "" {
			return nil
		}

		if d.IsDir() {
			return nil
		}

		file, err := os.OpenFile(path, os.O_RDWR, os.ModePerm)
		if err != nil {
			return errors.WithMessagef(err, "open file '%s'", path)
		}

		var spec kustomizationSpec

		// no need to handle non-YAML files as the decoder does the right thing
		err = yaml.NewDecoder(file).Decode(&spec)
		if err != nil {
			if errors.Cause(err) != io.EOF {
				return errors.WithMessagef(err, "decode '%s'", path)
			}
		}
		if strings.HasPrefix(spec.APIVersion, "kustomize.toolkit.fluxcd.io/") && spec.Kind == "Kustomization" {
			filePath = path
		}

		return nil
	})

	if err != nil {
		return "", errors.WithMessage(err, "walk directory")
	}

	return filePath, nil
}

func moveKustomizationToClusters(ctx context.Context, logger *log.Logger, srcPath, root, service, env, namespace string) error {
	destDir, err := kustomizationPath(root, env, namespace)
	if err != nil {
		return errors.WithMessage(err, "assemble kustomization path")
	}

	err = os.MkdirAll(destDir, os.ModePerm)
	if err != nil {
		return errors.WithMessagef(err, "create dest dir '%s'", destDir)
	}

	destPath, err := securejoin.SecureJoin(destDir, fmt.Sprintf("%s.yaml", service))
	if err != nil {
		return errors.WithMessage(err, "secure join destination path")
	}

	logger.WithContext(ctx).Infof("moveKustomizationToClusters: destPath '%s'", destPath)

	err = os.Rename(srcPath, destPath)
	if err != nil {
		return errors.WithMessagef(err, "move src '%s' to '%s'", srcPath, destPath)
	}

	return nil
}

func kustomizationPath(root, env, namespace string) (string, error) {
	kustomizationPath := root // start with root
	pathsToJoin := []string{
		"clusters",
		env,
		namespace,
	}
	var err error
	for _, p := range pathsToJoin {
		kustomizationPath, err = securejoin.SecureJoin(kustomizationPath, p)
		if err != nil {
			return "", errors.WithMessagef(err, "join '%s' to path", p)
		}
	}

	return kustomizationPath, nil
}
