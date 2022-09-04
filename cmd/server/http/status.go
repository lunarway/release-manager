package http

import (
	"context"
	"net/http"
	"time"

	"github.com/lunarway/release-manager/internal/flow"
	httpinternal "github.com/lunarway/release-manager/internal/http"
	"github.com/lunarway/release-manager/internal/log"
)

func status(payload *payload, flowSvc *flow.Service, logger *log.Logger) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		values := r.URL.Query()
		namespace := values.Get("namespace")
		service := values.Get("service")
		if emptyString(service) {
			requiredQueryError(w, logger, "service")
			return
		}

		ctx := r.Context()
		logger := logger.WithContext(ctx).WithFields("service", service, "namespace", namespace)
		s, err := flowSvc.Status(ctx, namespace, service)
		if err != nil {
			if ctx.Err() == context.Canceled {
				logger.Infof("http: status: get status cancelled: service '%s'", service)
				cancelled(w, logger)
				return
			}
			logger.Errorf("http: status: get status failed: service '%s': %v", service, err)
			unknownError(w, logger)
			return
		}

		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusOK)

		err = payload.encodeResponse(ctx, w, httpinternal.StatusResponse{
			DefaultNamespaces: s.DefaultNamespaces,
			Environments:      mapEnvironments(s.Environments),
		})
		if err != nil {
			logger.Errorf("http: status: service '%s': marshal response failed: %v", service, err)
		}
	}
}

func mapEnvironments(envs []flow.Environment) []httpinternal.Environment {
	var mapped []httpinternal.Environment
	for _, env := range envs {
		mapped = append(mapped, httpinternal.Environment{
			Name:                  env.Name,
			Message:               env.Message,
			Author:                env.Author,
			Tag:                   env.Tag,
			Committer:             env.Committer,
			Date:                  convertTimeToEpoch(env.Date),
			BuildUrl:              env.BuildURL,
			HighVulnerabilities:   env.HighVulnerabilities,
			MediumVulnerabilities: env.MediumVulnerabilities,
			LowVulnerabilities:    env.LowVulnerabilities,
		})
	}

	return mapped
}

func convertTimeToEpoch(t time.Time) int64 {
	return t.UnixNano() / int64(time.Millisecond)
}
