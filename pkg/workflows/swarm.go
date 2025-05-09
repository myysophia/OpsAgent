package workflows

import (
	"fmt"
	"os"
	"reflect"

	"github.com/feiskyer/swarm-go"
	"github.com/myysophia/OpsAgent/pkg/tools"
)

var (
	// auditFunc is a Swarm function that conducts a structured security audit of a Kubernetes Pod.
	trivyFunc = swarm.NewAgentFunction(
		"trivy",
		"Run trivy image scanning for a given image",
		func(args map[string]interface{}) (interface{}, error) {
			image, ok := args["image"].(string)
			if !ok {
				return nil, fmt.Errorf("image not provided")
			}

			result, err := tools.Trivy(image)
			if err != nil {
				return nil, err
			}

			return result, nil
		},
		[]swarm.Parameter{
			{Name: "image", Type: reflect.TypeOf(""), Required: true},
		},
	)

	// kubectlFunc is a Swarm function that runs kubectl command.
	kubectlFunc = swarm.NewAgentFunction(
		"kubectl",
		"Run kubectl command",
		func(args map[string]interface{}) (interface{}, error) {
			command, ok := args["command"].(string)
			if !ok {
				return nil, fmt.Errorf("command not provided")
			}

			result, err := tools.Kubectl(command)
			if err != nil {
				return nil, err
			}

			return result, nil
		},
		[]swarm.Parameter{
			{Name: "command", Type: reflect.TypeOf(""), Required: true},
		},
	)

	pythonFunc = swarm.NewAgentFunction(
		"python",
		"Run python code",
		func(args map[string]interface{}) (interface{}, error) {
			code, ok := args["code"].(string)
			if !ok {
				return nil, fmt.Errorf("code not provided")
			}

			result, err := tools.PythonREPL(code)
			if err != nil {
				return nil, err
			}

			return result, nil
		},
		[]swarm.Parameter{
			{Name: "code", Type: reflect.TypeOf(""), Required: true},
		},
	)
)

// NewSwarm creates a new Swarm client.
func NewSwarm() (*swarm.Swarm, error) {
	apiKey := os.Getenv("OPENAI_API_KEY")
	if apiKey != "" {
		baseURL := os.Getenv("OPENAI_API_BASE")
		if baseURL == "" {
			return swarm.NewSwarm(swarm.NewOpenAIClient(apiKey)), nil
		}

		// OpenAI compatible LLM
		return swarm.NewSwarm(swarm.NewOpenAIClientWithBaseURL(apiKey, baseURL)), nil
	}

	azureAPIKey := os.Getenv("AZURE_OPENAI_API_KEY")
	azureAPIBase := os.Getenv("AZURE_OPENAI_API_BASE")
	azureAPIVersion := os.Getenv("AZURE_OPENAI_API_VERSION")
	if azureAPIVersion == "" {
		azureAPIVersion = "2025-02-01-preview"
	}
	if azureAPIKey != "" && azureAPIBase != "" {
		return swarm.NewSwarm(swarm.NewAzureOpenAIClient(azureAPIKey, azureAPIBase, azureAPIVersion)), nil
	}

	return nil, fmt.Errorf("OPENAI_API_KEY or AZURE_OPENAI_API_KEY is not set")
}
