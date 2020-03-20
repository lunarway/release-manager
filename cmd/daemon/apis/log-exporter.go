package apis

import (
	"context"
	"net/http"

	"github.com/lunarway/release-manager/internal/log"
)

type LogExporter struct {
	Log *log.Logger
}

func (f *LogExporter) Send(_ context.Context, _ *http.Client, message Message) error {
	f.Log.With("FluxMessage", message).Infof("flux message exported")
	return nil
}
