package kubectl

import (
	"fmt"
	"os/exec"
)

type Executor struct{}

func NewExecutor() *Executor {
	return &Executor{}
}

func (e *Executor) ExecuteCommand(command string) (string, error) {
	cmd := exec.Command("sh", "-c", command)
	output, err := cmd.CombinedOutput()
	if err != nil {
		return "", fmt.Errorf("failed to execute command: %w, output: %s", err, string(output))
	}
	return string(output), nil
}

func (e *Executor) SwitchContext(context string) (string, error) {
	command := fmt.Sprintf("kubectl config use-context %s", context)
	return e.ExecuteCommand(command)
}
