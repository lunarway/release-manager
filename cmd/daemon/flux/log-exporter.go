package flux

import (
	"context"

	"github.com/lunarway/release-manager/internal/log"
)

type LogExporter struct {
	Log *log.Logger
}

func (f *LogExporter) Send(_ context.Context, message Message) error {
	f.Log.With("FluxMessage", message).Infof("flux message exported")
	return nil
}
