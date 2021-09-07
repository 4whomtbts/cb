package main

import (
	"circuitbreaker_slave/config"
	"circuitbreaker_slave/exporter"
	"circuitbreaker_slave/master"
	"circuitbreaker_slave/types"
	"circuitbreaker_slave/util"
	"context"
	"encoding/json"
	"fmt"
	ctypes "github.com/docker/docker/api/types"
	"github.com/docker/docker/client"
	"github.com/google/logger"
	"gopkg.in/yaml.v2"
	"io/ioutil"
	"net/http"
	"time"
)

func main() {
	var config *config.CircuitBreakerConfig
	configFile, err := ioutil.ReadFile("./circuitbreaker.yaml")
	if err != nil {
		panic("설정파일이 존재하지 않습니다")
	}
	err = yaml.Unmarshal(configFile, &config)
	if err != nil {
		panic(fmt.Sprintf("Failed to parse config file %s", err.Error()))
	}
	fmt.Println(fmt.Sprintf("Circuitbreaker 노드를 실행합니다. 노드타입 %s", config.Type))

	if config.Type == "master" {
		m := &master.Master{
			MailSender: util.NewMailSender(config.NodeName, config.Email, config.EmailPassword, config.EmailReceivers),
		}
		m.Start(config.Nodes, config.HealthCheckIntervalSec, config.CircuitBreakerIntervalSec)
	} else if config.Type == "node" {
		http.HandleFunc("/handshake", func(w http.ResponseWriter, req *http.Request) {
			w.WriteHeader(200)
			var metricServices = make([]*types.MetricService, len(config.Exporters))
			logger.Infof("[핸드셰이크] exporters=%v", config.Exporters)
			for i, metricService := range config.Exporters {
				metricServices[i] = &types.MetricService{
					Name:     metricService.Name,
					Endpoint: metricService.Url,
				}
			}
			node := &types.Node{
				Name:           config.NodeName,
				Endpoint:       "",
				Admins:         config.Admins,
				MetricServices: metricServices,
			}
			logger.Infof("[핸드셰이크] 핸드셰이크에 보낼 응답 %v", node)

			data, err := json.Marshal(node)
			if err != nil {
				w.WriteHeader(500)
				w.Write([]byte(err.Error()))
			}
			w.WriteHeader(200)

			w.Write(data)
		})

		http.HandleFunc("/reports/metric", func(w http.ResponseWriter, req *http.Request) {
			exporters := config.Exporters
			result := &types.MetricReport{
				NodeName:             config.NodeName,
				Endpoint:             "",
				MetricServiceReports: nil,
			}
			var services []*types.MetricServiceReport

			for _, currExporter := range exporters {
				metricReport := &types.MetricServiceReport{
					Name:      currExporter.Name,
					Endpoint:  currExporter.Url,
					IsHealthy: true,
					ErrLog:    "",
				}

				var concreteExporter exporter.Exporter
				switch currExporter.Label {
				case "node_exporter":
					concreteExporter = exporter.NewNodeExporter(currExporter.Url, currExporter.Config)
				case "dcgm_exporter":
					concreteExporter = exporter.NewDcgmExporter(currExporter.Url, currExporter.Config)
				}
				report := concreteExporter.GetExporterReport()
				if !report.Success {
					metricReport.IsHealthy = false
					metricReport.ErrLog = report.ErrLog
				}
				services = append(services, metricReport)
			}
			result.MetricServiceReports = services
			w.Header().Set("Content-Type", "application/json")
			data, err := json.Marshal(result)
			if err != nil {
				logger.Errorf("객체를 json 화 하는데 실패했습니다. %s", err.Error())
				w.WriteHeader(500)
				w.Write([]byte(err.Error()))
			}
			w.WriteHeader(200)
			w.Write(data)
		})

		http.HandleFunc("/reports/circuitbreaker", func(w http.ResponseWriter, req *http.Request) {
			exporters := config.Exporters
			var cbReports []*types.CircuitBreakerReport

			for _, currExporter := range exporters {
				cbReport := &types.CircuitBreakerReport{
					NodeName: config.NodeName,
					BrokenAt: "",
				}

				var concreteExporter exporter.Exporter
				switch currExporter.Label {
				case "node_exporter":
					concreteExporter = exporter.NewNodeExporter(currExporter.Url, currExporter.Config)
				case "dcgm_exporter":
					concreteExporter = exporter.NewDcgmExporter(currExporter.Url, currExporter.Config)
				}
				report := concreteExporter.GetExporterReport()
				if report.ShouldBreak {
					cbReport.BrokenAt = time.Now().String()
					var stoppedContainers []*types.Container
					ctx := context.Background()
					cli, err := client.NewClientWithOpts(client.FromEnv, client.WithAPIVersionNegotiation())
					if err != nil {
						panic(err)
					}
					containers, err := cli.ContainerList(ctx, ctypes.ContainerListOptions{})
					if err != nil {
						w.WriteHeader(500)
						w.Write([]byte("Failed to get container list"))
						return
					}

					for _, container := range containers {
						containerInfo := fmt.Sprintf("ID=%s, image=%s, name=%s, status=%s\n",
							container.ID, container.Image, container.Names, container.Status)

						fmt.Printf("Stopping container - %s", containerInfo)
						if err := cli.ContainerStop(ctx, container.ID, nil); err != nil {
							errLog := fmt.Sprintf("컨테이너(ID=%s) 정지에 실패하였습니다. %s", container.ID, err.Error())
							cbReport.Log = errLog
							logger.Errorf(errLog)
						}
						var joinedName string
						for _, name := range container.Names {
							joinedName += name
						}
						stoppedContainers = append(stoppedContainers, &types.Container{
							Id:   container.ID,
							Name: joinedName,
						})
						cbReport.Log += containerInfo
					}
					cbReport.StoppedContainers = stoppedContainers
				}
				cbReports = append(cbReports, cbReport)
			}
			w.Header().Set("Content-Type", "application/json")
			data, err := json.Marshal(cbReports)
			if err != nil {
				logger.Errorf("객체를 json 화 하는데 실패했습니다. %s", err.Error())
				w.WriteHeader(500)
				w.Write([]byte(err.Error()))
			}
			w.WriteHeader(200)
			w.Write(data)
		})
	} else {
		panic(fmt.Sprintf("타입 %s 은 존재하지 않습니다", config.Type))
	}
	err = http.ListenAndServe(fmt.Sprintf(":%d", config.Port), nil)
	if err != nil {
		panic(err.Error())
	}
}