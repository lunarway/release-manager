package flow

import "encoding/json"

type GenericEvent struct {
	Service     string `json:"service,omitempty"`
	Environment string `json:"environment,omitempty"`
	Namespace   string `json:"namespace,omitempty"`
	Branch      string `json:"branch,omitempty"`
	Actor       Actor  `json:"actor,omitempty"`
}

func (p *GenericEvent) Unmarshal(data []byte) error {
	return json.Unmarshal(data, p)
}