package command

import (
	"archive/zip"
	"bytes"
	"context"
	"fmt"
	"io/ioutil"
	"net/http"
	"os"
	"path"
	"path/filepath"

	"github.com/lunarway/release-manager/cmd/server/gpg"
	"github.com/lunarway/release-manager/internal/artifact"
	"github.com/lunarway/release-manager/internal/git"
	"github.com/lunarway/release-manager/internal/log"
	"github.com/lunarway/release-manager/internal/s3storage"
	"github.com/lunarway/release-manager/internal/tracing"
	"github.com/pkg/errors"
	"github.com/spf13/cobra"
)

func NewMigrate(startOptions *startOptions) *cobra.Command {
	var command = &cobra.Command{
		Use:   "migrate",
		Short: "migrate to s3 storage",
		RunE: func(c *cobra.Command, args []string) error {
			tracer, err := tracing.NewJaeger()
			if err != nil {
				return err
			}
			defer tracer.Close()

			// Import GPG Keys
			if startOptions.gitConfigOpts.SigningKey != "" {
				if len(*startOptions.gpgKeyPaths) < 1 {
					return errors.New("gpg signing key provided, but no import paths specified")
				}
				for _, p := range *startOptions.gpgKeyPaths {
					// lets just use flux' implementation on how to load keys
					keyfiles, err := gpg.ImportKeys(p, false)
					if err != nil {
						return fmt.Errorf("failed to import GPG key(s) from %s", p)
					}
					if keyfiles != nil {
						log.Infof("imported GPG key(s) from %s files %v", p, keyfiles)
					}
				}
			}
			gitSvc := git.Service{
				Tracer:            tracer,
				SSHPrivateKeyPath: startOptions.configRepo.SSHPrivateKeyPath,
				ConfigRepoURL:     startOptions.configRepo.ConfigRepo,
				Config:            startOptions.gitConfigOpts,
				ArtifactFileName:  startOptions.configRepo.ArtifactFileName,
			}

			s3storageSvc, err := s3storage.New(startOptions.s3storage.S3BucketName, tracer)
			if err != nil {
				return fmt.Errorf("failed setting up s3 storage: %w", err)
			}

			var _ = s3storageSvc

			ctx := context.Background()
			close, err := gitSvc.InitMasterRepo(ctx)
			if err != nil {
				return fmt.Errorf("failed initing master repo: %w", err)
			}
			defer close(ctx)

			readDir, closeReadDir, err := git.TempDir(ctx, tracer, "read")
			if err != nil {
				return fmt.Errorf("failed creating tmp read dir: %w", err)
			}
			defer closeReadDir(ctx)
			_, err = gitSvc.Clone(ctx, readDir)
			if err != nil {
				return fmt.Errorf("failed creating tmp read dir: %w", err)
			}

			serviceAndNamespace := map[string]string{}                // serviceName -> namespace
			serviceArtifacts := map[string]map[string]artifact.Spec{} // serviceName -> map of artifact IDs -> artifact.Spec
			for _, env := range []string{"dev", "staging", "prod"} {
				namespaceDirs, err := ioutil.ReadDir(path.Join(readDir, env, "releases"))
				if err != nil {
					return err
				}
				for _, namespaceDir := range namespaceDirs {
					if !namespaceDir.IsDir() {
						continue
					}

					serviceDirs, err := ioutil.ReadDir(path.Join(readDir, env, "releases", namespaceDir.Name()))
					if err != nil {
						return err
					}
					for _, serviceDir := range serviceDirs {
						if !serviceDir.IsDir() {
							continue
						}
						releasePath := path.Join(readDir, env, "releases", namespaceDir.Name(), serviceDir.Name(), "artifact.json")
						artifactSpec, err := artifact.Get(releasePath)
						if err != nil {
							return err
						}
						serviceAndNamespace[serviceDir.Name()] = namespaceDir.Name()
						if _, ok := serviceArtifacts[serviceDir.Name()]; !ok {
							serviceArtifacts[serviceDir.Name()] = make(map[string]artifact.Spec)
						}
						serviceArtifacts[serviceDir.Name()][artifactSpec.ID] = artifactSpec
					}
				}
			}

			var artifactResult []ArtifactInfo
			for service, artifactsMap := range serviceArtifacts {
				for artifactID, artifactSpec := range artifactsMap {
					exists, err := s3storageSvc.ArtifactExists(ctx, service, artifactID)
					if err != nil {
						return err
					}
					if exists {
						artifactResult = append(artifactResult, ArtifactInfo{
							ServiceName: service,
							ArtifactID:  artifactID,
							ExistsInS3:  true,
							Spec:        artifactSpec,
						})
						continue
					}

					specPath, _, artifactClose, err := gitSvc.ArtifactPaths(ctx, service, "dev", artifactSpec.Application.Branch, artifactID)
					if err != nil {
						if errors.Is(err, git.ErrArtifactNotFound) {
							artifactResult = append(artifactResult, ArtifactInfo{
								ServiceName:     service,
								ArtifactID:      artifactID,
								ExistsInS3:      false,
								InvalidArtifact: true,
							})
							continue
						}

						return err
					}

					artifactDir := path.Dir(specPath)
					zippedBytes, err := zipFiles(listFiles(artifactDir))
					if err != nil {
						return err
					}
					artifactClose(ctx)

					artifactResult = append(artifactResult, ArtifactInfo{
						ServiceName: service,
						ArtifactID:  artifactID,
						ExistsInS3:  false,
						ZippedBytes: zippedBytes,
						Spec:        artifactSpec,
					})

				}
			}

			for _, res := range artifactResult {
				if res.ExistsInS3 {
					fmt.Printf(" - %s - %s - exists in s3\n", res.ServiceName, res.ArtifactID)
					continue
				}
				if res.InvalidArtifact {
					fmt.Printf(" - %s - %s - invalid\n", res.ServiceName, res.ArtifactID)
					continue
				}
				fmt.Printf(" - %s - %s - length: %v\n", res.ServiceName, res.ArtifactID, len(res.ZippedBytes))
			}

			//s3storageSvc.CreateArtifact()
			//gitSvc.ArtifactPaths(ctx, service)

			return nil
		},
	}
	return command
}

type ArtifactInfo struct {
	ServiceName     string
	ArtifactID      string
	ExistsInS3      bool
	ZippedBytes     []byte
	Spec            artifact.Spec
	InvalidArtifact bool
}

func zipFiles(files []fileInfo) ([]byte, error) {
	buf := new(bytes.Buffer)
	w := zip.NewWriter(buf)
	for _, file := range files {
		f, err := w.Create(file.relativePath)
		if err != nil {
			return nil, err
		}
		fileContent, err := ioutil.ReadFile(file.fullPath)
		if err != nil {
			return nil, err
		}
		_, err = f.Write(fileContent)
		if err != nil {
			return nil, err
		}
	}

	err := w.Close()
	if err != nil {
		return nil, err
	}

	return buf.Bytes(), nil
}

func uploadFile(url string, fileContent []byte, md5 string) error {
	req, err := http.NewRequest(http.MethodPut, url, bytes.NewReader(fileContent))
	if err != nil {
		return err
	}

	req.Header.Set("Content-MD5", md5)

	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		return err
	}

	if resp.StatusCode != http.StatusOK {
		defer resp.Body.Close()
		body, err := ioutil.ReadAll(resp.Body)
		if err != nil {
			return fmt.Errorf("failed upload file to %s with status code %v and request id %v and and also got an error reading body %w", url, resp.StatusCode, resp.Header["X-Amz-Request-Id"], err)
		}
		return fmt.Errorf("failed upload file to %s with status code %v and request id %v and body %s", url, resp.StatusCode, resp.Header["X-Amz-Request-Id"], string(body))
	}

	return nil
}

func listFiles(path string) []fileInfo {
	var files []fileInfo
	err := filepath.Walk(path,
		func(filePath string, info os.FileInfo, err error) error {
			if err != nil {
				fmt.Printf("failed to walk dir: %v\n", err)
				return nil
			}
			if !info.IsDir() {
				fullPath, err := filepath.Abs(filePath)
				if err != nil {
					fmt.Printf("failed to generate absolute path for %s: %v\n", filePath, err)
					return nil
				}
				relativePath, err := filepath.Rel(path, filePath)
				if err != nil {
					fmt.Printf("failed to generate relative path for %s: %v\n", filePath, err)
					return nil
				}
				files = append(files, fileInfo{
					fullPath:     fullPath,
					relativePath: relativePath,
				})
			}
			return nil
		})
	if err != nil {
		fmt.Printf("failed to read dir: %v\n", err)
	}
	return files
}

type fileInfo struct {
	fullPath     string
	relativePath string
}
