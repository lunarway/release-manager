package grafana

import (
	"bytes"
	"encoding/json"
	"io/ioutil"
	"net/http"
	"time"

	"github.com/lunarway/release-manager/internal/log"
	"github.com/pkg/errors"
)

type AnnotateRequest struct {
	What string   `json:"what,omitempty"`
	Tags []string `json:"tags,omitempty"`
	Data string   `json:"data,omitempty"`
}

type AnnotateResponse struct {
	Message string `json:"message,omitempty"`
	Id      int64  `json:"id,omitempty"`
}

func Annotate(apiKey, baseURL string, body AnnotateRequest) error {
	client := &http.Client{
		Timeout: 10 * time.Second,
	}

	b := &bytes.Buffer{}
	err := json.NewEncoder(b).Encode(body)
	if err != nil {
		return err
	}

	req, err := http.NewRequest(http.MethodPost, baseURL+"/api/annotations/graphite", b)
	if err != nil {
		return err
	}
	req.Header.Set("Authorization", "Bearer "+apiKey)
	req.Header.Set("Content-Type", "application/json")
	resp, err := client.Do(req)
	if err != nil {
		return err
	}

	if resp.StatusCode != http.StatusOK {
		body, _ := ioutil.ReadAll(resp.Body)
		log.Infof("grafana: response body: %s", body)
		return errors.New("grafana: status code not ok")
	}
	var responseBody AnnotateResponse
	decoder := json.NewDecoder(resp.Body)
	err = decoder.Decode(&responseBody)
	if err != nil {
		return err
	}
	log.Infof("AnnoteResponse: message: %s, id: %d", responseBody.Message, responseBody.Id)
	return nil
}
