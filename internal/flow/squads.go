package flow

import (
	"io"
	"io/fs"
	"os"
	"path/filepath"
	"sort"
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

func squadsFromManifests(dir string) ([]string, error) {
	squadSet := make(map[string]struct{})

	err := filepath.WalkDir(dir, func(path string, d fs.DirEntry, walkErr error) error {
		if walkErr != nil {
			return errors.WithMessagef(walkErr, "walk dir '%s'", dir)
		}
		if d.IsDir() || !isManifestFile(path) {
			return nil
		}
		return scanManifestFile(path, squadSet)
	})
	if err != nil {
		return nil, errors.WithMessage(err, "walk manifest directory")
	}

	squads := make([]string, 0, len(squadSet))
	for squad := range squadSet {
		squads = append(squads, squad)
	}
	sort.Strings(squads)
	return squads, nil
}

func scanManifestFile(path string, squadSet map[string]struct{}) error {
	file, err := os.Open(path)
	if err != nil {
		return errors.WithMessagef(err, "open file '%s'", path)
	}
	defer file.Close()

	decoder := yaml.NewDecoder(file)
	for {
		var manifest manifestDocument
		err := decoder.Decode(&manifest)
		if err == io.EOF {
			return nil
		}
		if err != nil {
			return errors.WithMessagef(err, "decode '%s'", path)
		}
		collectManifestSquads(manifest, squadSet)
	}
}

func collectManifestSquads(manifest manifestDocument, squadSet map[string]struct{}) {
	addSquad(squadSet, manifest.Metadata.Labels["squad"])
	addSquad(squadSet, manifest.Spec.Template.Metadata.Labels["squad"])
	addSquad(squadSet, manifest.Spec.JobTemplate.Spec.Template.Metadata.Labels["squad"])

	for _, item := range manifest.Items {
		collectManifestSquads(item, squadSet)
	}
}

func addSquad(squadSet map[string]struct{}, squad string) {
	squad = strings.TrimSpace(squad)
	if squad == "" {
		return
	}
	squadSet[squad] = struct{}{}
}

func isManifestFile(path string) bool {
	switch strings.ToLower(filepath.Ext(path)) {
	case ".yaml", ".yml":
		return true
	default:
		return false
	}
}
