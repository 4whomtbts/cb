package service

type HealthCheckService struct {
	Nodes []*Node
}

func NewHealthCheckService(nodes []*Node) *HealthCheckService {
	return &HealthCheckService {
		nodes,
	}
}
