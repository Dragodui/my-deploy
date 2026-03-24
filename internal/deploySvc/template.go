package deploysvc

import (
	"encoding/json"
	"net/http"

	"github.com/dragodui/my-deploy/internal/shared/models"
)

type TemplateClient struct {
	baseURL string
	http    *http.Client
}

func NewTemplateClient(baseURL string) *TemplateClient {
	return &TemplateClient{
		baseURL: baseURL,
		http:    &http.Client{},
	}
}

func (c *TemplateClient) Get(id string) (*models.AppTemplate, bool) {
	resp, err := c.http.Get(c.baseURL + "/internal/templates/" + id)
	if err != nil {
		return nil, false
	}
	defer resp.Body.Close()

	var template models.AppTemplate
	if err := json.NewDecoder(resp.Body).Decode(&template); err != nil {
		return nil, false
	}

	return &template, true
}
