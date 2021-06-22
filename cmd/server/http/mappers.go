package http

import (
	"fmt"
	"time"

	"github.com/go-openapi/strfmt"
	"github.com/lunarway/release-manager/generated/http/models"
	"github.com/lunarway/release-manager/internal/artifact"
	"github.com/lunarway/release-manager/internal/intent"
	"github.com/lunarway/release-manager/internal/policy"
)

func mapIntent(i intent.Intent) *models.Intent {
	return &models.Intent{
		Promote: &models.IntentPromote{
			FromEnvironment: i.Promote.FromEnvironment,
		},
		ReleaseBranch: &models.IntentReleaseBranch{
			Branch: i.ReleaseBranch.Branch,
		},
		Rollback: &models.IntentRollback{
			PreviousArtifactID: i.Rollback.PreviousArtifactID,
		},
		Type: &i.Type,
	}
}

func mapBuildStage(stage artifact.Stage) *models.ArtifactStagesItems0 {
	data := stage.Data.(artifact.BuildData)
	return &models.ArtifactStagesItems0{
		ArtifactStageBuild: models.ArtifactStageBuild{
			ID:   string(stage.ID),
			Name: stage.Name,
			Data: &models.ArtifactStageBuildData{
				DockerVersion: data.DockerVersion,
				Image:         data.Image,
				Tag:           data.Tag,
			},
		},
	}
}

func mapTestStage(stage artifact.Stage) *models.ArtifactStagesItems0 {
	data := stage.Data.(artifact.TestData)
	return &models.ArtifactStagesItems0{
		ArtifactStageTest: models.ArtifactStageTest{
			ID:   string(stage.ID),
			Name: stage.Name,
			Data: &models.ArtifactStageTestData{
				URL: data.URL,
				Results: &models.ArtifactStageTestDataResults{
					Failed:  int64(data.Results.Failed),
					Passed:  int64(data.Results.Passed),
					Skipped: int64(data.Results.Skipped),
				},
			},
		},
	}
}

func mapPushStage(stage artifact.Stage) *models.ArtifactStagesItems0 {
	data := stage.Data.(artifact.PushData)
	return &models.ArtifactStagesItems0{
		ArtifactStagePush: models.ArtifactStagePush{
			ID:   string(stage.ID),
			Name: stage.Name,
			Data: &models.ArtifactStagePushData{
				DockerVersion: data.DockerVersion,
				Image:         data.Image,
				Tag:           data.Tag,
			},
		},
	}
}

func mapSnykCodeStage(stage artifact.Stage) *models.ArtifactStagesItems0 {
	data := stage.Data.(artifact.SnykCodeData)
	return &models.ArtifactStagesItems0{
		ArtifactStageSnykCode: models.ArtifactStageSnykCode{
			ID:   string(stage.ID),
			Name: stage.Name,
			Data: &models.ArtifactStageSnykCodeData{
				Language:        data.Language,
				SnykVersion:     data.SnykVersion,
				URL:             data.URL,
				Vulnerabilities: mapVulnerabilities(data.Vulnerabilities),
			},
		},
	}
}

func mapSnykDockerStage(stage artifact.Stage) *models.ArtifactStagesItems0 {
	data := stage.Data.(artifact.SnykDockerData)
	return &models.ArtifactStagesItems0{
		ArtifactStageSnykDocker: models.ArtifactStageSnykDocker{
			ID:   string(stage.ID),
			Name: stage.Name,
			Data: &models.ArtifactStageSnykDockerData{
				BaseImage:       data.BaseImage,
				Tag:             data.Tag,
				SnykVersion:     data.SnykVersion,
				URL:             data.URL,
				Vulnerabilities: mapVulnerabilities(data.Vulnerabilities),
			},
		},
	}
}

func mapVulnerabilities(v artifact.VulnerabilityResult) *models.Vulnerabilities {
	return &models.Vulnerabilities{
		High:   int64(v.High),
		Medium: int64(v.Medium),
		Low:    int64(v.Low),
	}
}

func mapArtifactToHTTP(a artifact.Spec) *models.Artifact {
	var httpStages []*models.ArtifactStagesItems0
	for _, stage := range a.Stages {
		switch stage.ID {
		case artifact.StageIDBuild:
			httpStages = append(httpStages, mapBuildStage(stage))
		case artifact.StageIDTest:
			httpStages = append(httpStages, mapTestStage(stage))
		case artifact.StageIDPush:
			httpStages = append(httpStages, mapPushStage(stage))
		case artifact.StageIDSnykCode:
			httpStages = append(httpStages, mapSnykCodeStage(stage))
		case artifact.StageIDSnykDocker:
			httpStages = append(httpStages, mapSnykDockerStage(stage))
		default:
			panic(fmt.Errorf("unknown stage id '%s' when mapping to HTTP model", stage.ID))
		}
	}

	return &models.Artifact{
		Application: mapRepository(a.Application),
		Ci: &models.ArtifactCi{
			End:    strfmt.Date(a.CI.End),
			JobURL: a.CI.JobURL,
			Start:  strfmt.Date(a.CI.Start),
		},
		ID:        a.ID,
		Namespace: a.Namespace,
		Service:   a.Service,
		Shuttle: &models.ArtifactShuttle{
			Plan:           mapRepository(a.Shuttle.Plan),
			ShuttleVersion: a.Shuttle.ShuttleVersion,
		},
		Squad:  a.Squad,
		Stages: httpStages,
	}
}

func mapRepository(r artifact.Repository) *models.Repository {
	return &models.Repository{
		Branch:         r.Branch,
		Sha:            r.SHA,
		AuthorName:     r.AuthorName,
		AuthorEmail:    r.AuthorEmail,
		CommitterName:  r.CommitterName,
		CommitterEmail: r.CommitterEmail,
		Message:        r.Message,
		Name:           r.Name,
		URL:            r.URL,
		Provider:       r.Provider,
	}
}

func mapAutoReleasePolicies(policies []policy.AutoReleasePolicy) []*models.GetPoliciesResponseAutoReleasesItems0 {
	h := make([]*models.GetPoliciesResponseAutoReleasesItems0, len(policies))
	for i, p := range policies {
		h[i] = &models.GetPoliciesResponseAutoReleasesItems0{
			ID:          p.ID,
			Branch:      p.Branch,
			Environment: p.Environment,
		}
	}
	return h
}

func mapBranchRestrictionPolicies(policies []policy.BranchRestriction) []*models.GetPoliciesResponseBranchRestrictionsItems0 {
	h := make([]*models.GetPoliciesResponseBranchRestrictionsItems0, len(policies))
	for i, p := range policies {
		h[i] = &models.GetPoliciesResponseBranchRestrictionsItems0{
			ID:          p.ID,
			Environment: p.Environment,
			BranchRegex: p.BranchRegex,
		}
	}
	return h
}

func convertTimeToEpoch(t time.Time) int64 {
	return t.UnixNano() / int64(time.Millisecond)
}
