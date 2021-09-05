package types

type MetricReport struct {
	NodeName string `json:"node_name"`
	Endpoint string `json:"end_point"`
	MetricServiceReports []*MetricServiceReport `json:"metric_service_reports"`
}

type MetricServiceReport struct {
	Name string `json:"name"`
	Endpoint string `json:"endpoint"`
	IsHealthy bool `json:"is_healthy"`
	ErrLog string `json:"err_log"`
}

type CircuitBreakerReport struct {
	NodeName string `json:"node_name"`
	Log string `json:"log"`
	BrokenAt string `json:"broken_at"`
	BrokenMetrics []*BrokenMetric
	StoppedContainers []*Container
}

type BrokenMetric struct {
	Name string `json:"name"`
	Reason string `json:"reason"`
	ErrLog string `json:"string"`
}

type Container struct {
	Id string `json:"id"`
	Name string `json:"name"`
}

type Node struct {
	Name string `json:"name"`
	Endpoint string `json:"endpoint"`
	Admins []*Admin `json:"admins"`
	MetricServices []*MetricService `json:"metric_services"`
}

type MetricService struct {
	Name string `json:"name"`
	Endpoint string `json:"endpoint"`
}

type Admin struct {
	Name string `yaml:"name"`
	Email string `yaml:"email"`
	Phone string `yaml:"phone"`
}


