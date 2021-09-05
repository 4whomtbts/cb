package circuitbreaker

import (
	"context"
	"fmt"
	"github.com/docker/docker/api/types"
	"github.com/docker/docker/client"
	"time"
)

type CircuitBreaker interface {
	BreakCircuit() (*BreakResult, *BreakError)
}

type BreakError struct {
	Message string
}

func NewBreakError(message string) *BreakError {
	return &BreakError{
		Message: message,
	}
}

type BreakResult struct {
	Causation string
	Result string
	StartedAt int64
	ExporterName string
	ExporterUrl string
}

func NewBreakResult(result string, startedAt int64) *BreakResult {
	return &BreakResult{
		Result: result,
		StartedAt: startedAt,
	}
}

type SoftCircuitBreaker struct {
	dockerContext context.Context
	dockerCli *client.Client
}

func NewSoftCircuitBreaker() *SoftCircuitBreaker {
	softCircuitBreaker := &SoftCircuitBreaker{}
	softCircuitBreaker.dockerContext = context.Background()
	cli, err := client.NewClientWithOpts(client.FromEnv, client.WithAPIVersionNegotiation())
	if err != nil {
		panic(fmt.Sprintf("softCircuitBreaker failed to create cli for docker! error was %s", err.Error()))
	}
	softCircuitBreaker.dockerCli = cli
	return softCircuitBreaker
}

func (sc *SoftCircuitBreaker) BreakCircuit() (*BreakResult, *BreakError){
	startedAt := time.Now().Unix()
	containers, err := sc.dockerCli.ContainerList(sc.dockerContext, types.ContainerListOptions{})
	if err != nil {

	}
	partialFailure := false
	errorMessage := ""
	stoppedContainers := ""

	for _, container := range containers {
		containerInfo := fmt.Sprintf("ID=%s, image=%s, name=%s, status=%s",
			container.ID, container.Image, container.Names, container.Status)

		fmt.Printf("Stopping container - %s", containerInfo)
		if err := sc.dockerCli.ContainerStop(sc.dockerContext, container.ID, nil); err != nil {
			partialFailure = true
			errorMessage += fmt.Sprintf("failed to stop container %s. %s\n", container.ID, err.Error())
			continue
		}
		stoppedContainers += fmt.Sprintf("%s\n", containerInfo)
	}

	if (partialFailure) {
		return nil, NewBreakError(errorMessage)
	}
	return NewBreakResult(stoppedContainers, startedAt), nil
}