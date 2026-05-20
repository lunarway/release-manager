package flow

import (
	"io"
	"io/fs"
	"os"
	"path/filepath"
	"strings"

	"github.com/pkg/errors"
	yaml "gopkg.in/yaml.v3"
)

type manifestMetadata struct {
	Labels map[string]string `yaml:"labels"`
}

type manifestTemplate struct {
	Metadata manifestMetadata `yaml:"metadata"`
}

type manifestJobTemplate struct {
	Spec struct {
		Template manifestTemplate `yaml:"template"`
	} `yaml:"spec"`
}

type manifestSpec struct {
	Template    manifestTemplate    `yaml:"template"`
	JobTemplate manifestJobTemplate `yaml:"jobTemplate"`
}

type manifestDocument struct {
	Metadata manifestMetadata   `yaml:"metadata"`
	Spec     manifestSpec       `yaml:"spec"`
	Items    []manifestDocument `yaml:"items"`
}

func squadFromManifests(dir string) (string, error) {
	var squad string

	err := filepath.WalkDir(dir, func(path string, d fs.DirEntry, walkErr error) error {
		if walkErr != nil {
			return errors.WithMessagef(walkErr, "walk dir '%s'", dir)
		}
		if d.IsDir() || !isManifestFile(path) {
			return nil
		}
		squadInFile, err := scanManifestFile(path)
		if err != nil {
			return err
		}
		if squadInFile == "" {
			return nil
		}
		squad = squadInFile
		return filepath.SkipAll
	})
	if err != nil {
		return "", errors.WithMessage(err, "walk manifest directory")
	}
	return squad, nil
}

func scanManifestFile(path string) (string, error) {
	file, err := os.Open(path)
	if err != nil {
		return "", errors.WithMessagef(err, "open file '%s'", path)
	}
	defer file.Close()

	decoder := yaml.NewDecoder(file)
	for {
		var manifest manifestDocument
		err := decoder.Decode(&manifest)
		if err == io.EOF {
			return "", nil
		}
		if err != nil {
			return "", errors.WithMessagef(err, "decode '%s'", path)
		}
		squad := squadFromManifestDocument(manifest)
		if squad != "" {
			return squad, nil
		}
	}
}

func squadFromManifestDocument(manifest manifestDocument) string {
	if squad := normalizeSquad(manifest.Metadata.Labels["squad"]); squad != "" {
		return squad
	}
	if squad := normalizeSquad(manifest.Spec.Template.Metadata.Labels["squad"]); squad != "" {
		return squad
	}
	if squad := normalizeSquad(manifest.Spec.JobTemplate.Spec.Template.Metadata.Labels["squad"]); squad != "" {
		return squad
	}

	for _, item := range manifest.Items {
		if squad := squadFromManifestDocument(item); squad != "" {
			return squad
		}
	}
	return ""
}

func normalizeSquad(squad string) string {
	return strings.TrimSpace(squad)
}

func isManifestFile(path string) bool {
	switch strings.ToLower(filepath.Ext(path)) {
	case ".yaml", ".yml":
		return true
	default:
		return false
	}
}
