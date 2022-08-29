package flux2notifications

import (
	"io"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/lunarway/release-manager/internal/log"
	"github.com/stretchr/testify/assert"
	"go.uber.org/zap/zapcore"
)

// TestWebhookForAlerts tests if /flux2-alerts can handle requests from flux2-notification-controller
func TestWebhookForAlerts(t *testing.T) {
	//arrange
	setupTest()
	var (
		eventAsJson = "{\"involvedObject\": {\"kind\":\"GitRepository\",\"namespace\":\"flux-system\",\"name\":\"flux-system\",\"uid\":\"cc4d0095-83f4-4f08-98f2-d2e9f3731fb9\",\"apiVersion\":\"source.toolkit.fluxcd.io/v1beta2\", \"resourceVersion\":\"56921\"},\"severity\":\"info\",\"timestamp\":\"2006-01-02T15:04:05Z\",\"message\":\"Fetched revision: main/731f7eaddfb6af01cb2173e18f0f75b0ba780ef1\",\"reason\":\"info\",\"reportingController\":\"source-controller\",\"reportingInstance\":\"source-controller-7c7b47f5f-8bhrp\"}"
		request     = httptest.NewRequest(http.MethodPost, "/webhook/flux2-alerts", strings.NewReader(eventAsJson))
		w           = httptest.NewRecorder()
	)

	//act
	HandleEventFromFlux2(w, request)
	response := w.Result()
	defer response.Body.Close()
	_, err := io.ReadAll(response.Body)

	//assert
	assert.NoError(t, err, "HandleEventFromFlux2 could not handle request")
	assert.Equal(t, http.StatusOK, response.StatusCode)
}

func setupTest() {
	log.Init(&log.Configuration{
		Level: log.Level{
			Level: zapcore.DebugLevel,
		},
		Development: true,
	})
}
