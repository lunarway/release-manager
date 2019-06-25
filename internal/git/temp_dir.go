package git

import (
	"context"
	"fmt"
	"io/ioutil"
	"os"

	"github.com/lunarway/release-manager/internal/log"
	"github.com/opentracing/opentracing-go"
)

// TempDir returns a temporary directory with provided prefix.
// The first return argument is the path. The second is a close function to
// remove the path.
func TempDir(ctx context.Context, tracer opentracing.Tracer, prefix string) (string, func(context.Context), error) {
	span, ctx := opentracing.StartSpanFromContextWithTracer(ctx, tracer, fmt.Sprintf("create temp dir for '%s'", prefix))
	defer span.Finish()
	path, err := ioutil.TempDir("", prefix)
	if err != nil {
		return "", func(context.Context) {}, err
	}
	return path, func(ctx context.Context) {
		span, ctx := opentracing.StartSpanFromContextWithTracer(ctx, tracer, fmt.Sprintf("clean temp dir for '%s'", prefix))
		defer span.Finish()
		err := os.RemoveAll(path)
		if err != nil {
			log.Errorf("Removing temporary directory failed: path '%s': %v", path, err)
		}
	}, nil
}
