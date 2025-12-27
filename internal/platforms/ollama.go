package platforms

import "github.com/ollama/ollama/api"

type OllamaPlatform struct {
	client *api.Client
}

func NewOllamaPlatform() *OllamaPlatform {
	client, err := api.ClientFromEnvironment()
	if err != nil {
		panic("failed to create Ollama client: " + err.Error())
	}
	return &OllamaPlatform{
		client: client,
	}
}

func (o *OllamaPlatform) Client() *api.Client { return o.client }
