package flow

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestSquadFromManifests(t *testing.T) {
	t.Run("returns the first squad found in manifest walk order", func(t *testing.T) {
		dir := t.TempDir()

		require.NoError(t, os.WriteFile(filepath.Join(dir, "a-deployment.yaml"), []byte(`
apiVersion: apps/v1
kind: Deployment
metadata:
  labels:
    squad: alpha
spec:
  template:
    metadata:
      labels:
        app: demo
`), 0o600))

		require.NoError(t, os.WriteFile(filepath.Join(dir, "b-job.yaml"), []byte(`
apiVersion: batch/v1
kind: Job
spec:
  template:
    metadata:
      labels:
        squad: beta
`), 0o600))

		require.NoError(t, os.WriteFile(filepath.Join(dir, "c-cronjob.yaml"), []byte(`
apiVersion: batch/v1
kind: CronJob
spec:
  jobTemplate:
    spec:
      template:
        metadata:
          labels:
            squad: gamma
---
apiVersion: v1
kind: Service
metadata:
  labels:
    squad: alpha
`), 0o600))

		require.NoError(t, os.WriteFile(filepath.Join(dir, "README.txt"), []byte("not a manifest"), 0o600))

		actual, err := squadFromManifests(dir)

		require.NoError(t, err)
		assert.Equal(t, "alpha", actual)
	})

	t.Run("returns an error for invalid yaml manifests", func(t *testing.T) {
		dir := t.TempDir()
		require.NoError(t, os.WriteFile(filepath.Join(dir, "invalid.yaml"), []byte("metadata: ["), 0o600))

		_, err := squadFromManifests(dir)

		require.Error(t, err)
		assert.Contains(t, err.Error(), "decode")
	})

	t.Run("short circuits before later invalid manifests once a squad is found", func(t *testing.T) {
		dir := t.TempDir()

		require.NoError(t, os.WriteFile(filepath.Join(dir, "a-valid.yaml"), []byte(`
apiVersion: apps/v1
kind: Deployment
metadata:
  labels:
    squad: alpha
`), 0o600))
		require.NoError(t, os.WriteFile(filepath.Join(dir, "z-invalid.yaml"), []byte("metadata: ["), 0o600))

		actual, err := squadFromManifests(dir)

		require.NoError(t, err)
		assert.Equal(t, "alpha", actual)
	})
}
