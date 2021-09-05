package master

import (
	"circuitbreaker_slave/types"
	"circuitbreaker_slave/util"
	"encoding/json"
	"fmt"
	"github.com/google/logger"
	"io/ioutil"
	"net/http"
	"time"
)

type Master struct {
	MailSender *util.MailSender
}

func (m *Master) handShakeNodes(endpoints []string) []*types.Node {
	nodes := make([]*types.Node, len(endpoints))

	defaultTimeout := 10 * time.Second
	httpClient := &http.Client{
		Timeout: defaultTimeout,
	}
	for i, endpoint := range endpoints {
		logger.Infof("노드(엔드포인트=%s) 와 핸드셰이크를 시작합니다", endpoint)
		resp, err := httpClient.Get(endpoint + "/handshake")
		if err != nil {
			logger.Errorf("노드 엔드포인트 %s 와의 핸드셰이킹에 실패했습니다 : %v", endpoint, err)
			resp.Body.Close()
			continue
		}

		body, err := ioutil.ReadAll(resp.Body)
		if err != nil {
			logger.Errorf("핸드셰이킹 응답에서 Body 를 읽을 수 없습니다 : %v", err)
		}
		var node *types.Node
		err = json.Unmarshal(body, &node)
		if err != nil {
			panic(fmt.Sprintf("핸드셰이킹 응답 형식이 잘못되었습니다 : %v", err))
		}

		node.Endpoint = endpoint
		nodes[i] = node
		resp.Body.Close()
	}
	logger.Infof("핸드셰이크에 성공했습니다. 노드 정보 %v", nodes)
	return nodes
}

func requestMetricReports(nodes []*types.Node) []*types.MetricReport {
	reports := make([]*types.MetricReport, len(nodes))

	defaultTimeout := 15 * time.Second
	httpClient := &http.Client{
		Timeout: defaultTimeout,
	}

	for i, node := range nodes {
		resp, err := httpClient.Get(node.Endpoint + "/reports/metric")
		if err != nil {
			logger.Errorf("엔드포인트 %s 에대한 레포트 요청이 실패했습니다 %v", node.Endpoint, err)
		}
		body, err := ioutil.ReadAll(resp.Body)
		if err != nil {
			logger.Errorf("레포트 응답에서 Body 를 읽을 수 없습니다 : %v", err)
			resp.Body.Close()
			continue
		}

		if resp.StatusCode != 200 {
			resp.Body.Close()
		}

		var report *types.MetricReport
		err = json.Unmarshal(body, &report)
		if err != nil {
			logger.Errorf(fmt.Sprintf("노드(%v) 의 레포트 응답 형식이 잘못되었습니다 : %v", node, err))
		}
		report.Endpoint = node.Endpoint
		reports[i] = report
	}
	return reports
}

func requestCircuitBreakerReport(nodes []*types.Node) [][]*types.CircuitBreakerReport {
	var reports [][]*types.CircuitBreakerReport

	defaultTimeout := 15 * time.Second
	httpClient := &http.Client{
		Timeout: defaultTimeout,
	}
	for _, node := range nodes {
		resp, err := httpClient.Get(node.Endpoint + "/reports/circuitbreaker")
		if err != nil {
			logger.Errorf("엔드포인트 %s 에대한 레포트 요청이 실패했습니다 %v", node.Endpoint, err)
		}

		body, err := ioutil.ReadAll(resp.Body)
		if err != nil {
			logger.Errorf("레포트 응답에서 Body 를 읽을 수 없습니다 : %v", err)
			resp.Body.Close()
			continue
		}

		if resp.StatusCode != 200 {
			logger.Errorf("노드 %s 에서 레포트를 가져오는데 실패했습니다. statusCode=%d, body=%s", resp.StatusCode, resp.Body)
			resp.Body.Close()
			continue
		}

		var report []*types.CircuitBreakerReport
		err = json.Unmarshal(body, &report)
		logger.Infof("data %s", string(body))
		if err != nil {
			logger.Errorf(fmt.Sprintf("노드(%v) 의 레포트 응답 형식이 잘못되었습니다 : %v", node, err.Error()))
		}
		reports = append(reports, report)
	}
	return reports
}

func buildMetricReportMailContent(reports []*types.MetricReport) string {
	var analysisReport string
	for _, report := range reports {
		analysisReport += fmt.Sprintf(
			"<h2>노드(노드명=%s, 엔드포인트=%s) 레포트</h2>", report.NodeName, report.Endpoint)
		for _, metricReport := range report.MetricServiceReports {
			analysisReport += fmt.Sprintf(
				"<h3>메트릭(얼라이어스=%s, 엔드포인트=%s)</h3>", metricReport.Name, metricReport.Endpoint)
			if !metricReport.IsHealthy {
				analysisReport += fmt.Sprintf(
					"<p>메트릭 수집에 실패했습니다</p><span>에러로그 : %s </span><br><br>", metricReport.ErrLog)
				continue
			}
			analysisReport += fmt.Sprintf("<p>메트릭이 정상적으로 수집되고 있습니다</p><br>")
		}
	}
	return analysisReport
}

func buildCircuitBreakerReportMailContent(nodeCbReports [][]*types.CircuitBreakerReport) string {
	var analysisReport string
	analysisReport += "<h1> 서킷브레이커 발동 내역 </h1><br>"
	for _, nodeCbReport := range nodeCbReports {
		for _, report := range nodeCbReport {
			if report.BrokenAt == "" {
				continue
			}
			analysisReport += fmt.Sprintf(
				"<h2>노드(노드명=%s) 레포트</h2>", report.NodeName)
			for i, stoppedContainer := range report.StoppedContainers {
				if i == 0 {
					analysisReport += fmt.Sprintf("<h3>중지된 컨테이너 목록</h3>")
				}
				analysisReport += fmt.Sprintf("<p>컨테이너 ID: %s, 컨테이너명: %s </p>",
					stoppedContainer.Id, stoppedContainer.Name)
			}
			analysisReport += fmt.Sprintf("서킷브레이킹 로그 : %s", report.Log)
		}
	}
	return analysisReport
}

func (m *Master) doHealthCheck(nodes []*types.Node, healthCheckIntervalSec int) {
	for ;; {
		mailContent := buildMetricReportMailContent(requestMetricReports(nodes))
		logger.Infof("doHealthCheck mail : %s", mailContent)
		m.MailSender.Send("[서킷브레이커] Healthcheck 보고서", mailContent)
		time.Sleep(time.Duration(healthCheckIntervalSec) * time.Second)
	}
}

func (m *Master) doCircuitBrokenCheck(nodes []*types.Node, circuitBrokenCheckIntervalSec int) {
	for ;; {
		brokenHappened := false
		nodeCbReports := requestCircuitBreakerReport(nodes)
		for _, nodeCbReport := range nodeCbReports {
			for _, report := range nodeCbReport {
				if report.BrokenAt != "" {
					brokenHappened = true
					break
				}
			}
		}
		if brokenHappened {
			mailContent := buildCircuitBreakerReportMailContent(nodeCbReports)
			logger.Infof("doCircuitBrokenCheck mail : %s", mailContent)
			m.MailSender.Send("[서킷브레이커] CircuitBreaking 보고서", mailContent)
		}
		time.Sleep(time.Duration(circuitBrokenCheckIntervalSec) * time.Second)
	}
}

func (m *Master) Start(nodeEndpoints []string, healthCheckIntervalSec int, circuitBrokenCheckIntervalSec int) {
	nodes := m.handShakeNodes(nodeEndpoints)
	go m.doHealthCheck(nodes, healthCheckIntervalSec)
	go m.doCircuitBrokenCheck(nodes, healthCheckIntervalSec)
	for ;; {}
}