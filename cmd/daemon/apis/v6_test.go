package apis

import (
	"bytes"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestHandleV6(t *testing.T) {
	fakeExporter := &FakeExporter{}
	config := NewFakeConfig()
	config.Set("github_url", "https://github.com")

	formatter, _ := NewDefaultFormatter(config)

	apiConfig := APIConfig{
		Server:    http.NewServeMux(),
		Exporter:  []Exporter{fakeExporter},
		Formatter: formatter,
	}

	HandleV6(apiConfig)

	event := test_utils.NewFluxSyncEvent()
	data, _ := json.Marshal(event)
	req, _ := http.NewRequest("POST", "http://127.0.0.1:3030/v6/events", bytes.NewBuffer(data))

	recorder := httptest.NewRecorder()
	apiConfig.Server.ServeHTTP(recorder, req)
	resp := recorder.Result()
	assert.Equal(t, 200, resp.StatusCode)

	formatted := formatter.FormatEvent(event, fakeExporter)
	assert.Equal(t, formatted.Title, fakeExporter.Sent[0].Title, formatted.Title)
	assert.Equal(t, formatted.Body, fakeExporter.Sent[0].Body, formatted.Body)
}
