package service

type Node struct {
	Name string `json:"name"`
	Endpoint string `json:"endpoint"`
	Admins []*Admin `json:"admins"`
	MetricServices []*MetricService `json:"metric_services"`
}

type MetricService struct {
	Name string `json:"name"`
	Endpoint string `json:"endpoint"`
	IsHealthy bool `json:"is_healthy"`
	ErrLog string `json:"err_log"`
}

type Admin struct {
	Name string `json:"name"`
	Email string `json:"email"`
	Phone string `json:"phone"`
}