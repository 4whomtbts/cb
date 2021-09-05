package master

import "circuitbreaker_slave/types"

type HealthCheckService struct {
	Nodes []*types.Node
}

func NewHealthCheckService(nodes []*types.Node) *HealthCheckService {
	return &HealthCheckService {
		nodes,
	}
}