package watcher

import (
	"circuitbreaker_slave/circuitbreaker"
	"circuitbreaker_slave/config"
	"circuitbreaker_slave/exporter"
	"circuitbreaker_slave/types"
	"circuitbreaker_slave/util"
	"fmt"
	log "github.com/sirupsen/logrus"
	"time"
)

type Watcher struct {
	mailSender *util.MailSender
	circuitBreaker circuitbreaker.CircuitBreaker
	exporters []exporter.Exporter
	watchIntervalSec int
	nodeName string
}

func NewWatcher(mailSender *util.MailSender, watchIntervalSec int, nodeName string, exporters []config.Exporter) *Watcher {

	watcher := &Watcher {
		nodeName: nodeName,
		mailSender: mailSender,
	}
	var newExporters []exporter.Exporter
	for _, currExporter := range exporters {
		if currExporter.Label == "node_exporter" {
			newExporters = append(newExporters, exporter.NewNodeExporter(currExporter.Config))
		} else if currExporter.Label == "dcgm_exporter" {

		} else {
			panic(fmt.Sprintf("exporter label [%s] doesn't exists", currExporter.Label))
		}
	}
	watcher.exporters = newExporters
	watcher.watchIntervalSec = watchIntervalSec
	return watcher
}

func (watcher *Watcher) getMetricsReport() *types.MetricReport {
	metricReport := &types.MetricReport{
		NodeName: watcher.nodeName,
		MetricServiceReports: make([]*types.MetricServiceReport, len(watcher.exporters)),
	}
	for _, e := range watcher.exporters {
		report := e.GetExporterReport()
		metricServiceReport := &types.MetricServiceReport{
			Name:      watcher.nodeName,
		}
		if !report.Success {
			metricServiceReport.IsHealthy = false
			metricServiceReport.ErrLog = report.ErrLog
			metricReport.MetricServiceReports = append(metricReport.MetricServiceReports, metricServiceReport)
			continue
		}
	}
	return metricReport
}

func (watcher *Watcher) getCircuitBreakersReport() *types.CircuitBreakerReport {
	circuitbreakerReport := &types.CircuitBreakerReport{
		NodeName:          watcher.nodeName,
		Log:               "",
		BrokenAt:          "",
		StoppedContainers: nil,
	}
	for _, e := range watcher.exporters {
		report := e.GetExporterReport()
		if report.ShouldBreak {
			breakResult, breakErr := watcher.circuitBreaker.BreakCircuit()
			brokenMetric := &types.BrokenMetric {
				Name: report.ExporterName,
				Reason: "",
			}
			if breakErr != nil {
				brokenMetric.ErrLog = breakErr.Message
				circuitbreakerReport.BrokenMetrics =
					append(circuitbreakerReport.BrokenMetrics, brokenMetric)
			} else {
				message := fmt.Sprintf("서킷브레이크 발동사유: %s, 발동결과: %s\n", report.Reason, breakResult.Result)
				brokenMetric.Reason = message
				circuitbreakerReport.BrokenMetrics =
					append(circuitbreakerReport.BrokenMetrics, brokenMetric)
			}
		}
	}
	return circuitbreakerReport
}

func (watcher *Watcher) Watch() {
	circuitBreakerExecuted := false
	for ;; {
		for _, e := range watcher.exporters {
			report := e.GetExporterReport()
			if !report.Success {
				log.Errorf(
					"Failed to get report from %s(url=%s)", report.ExporterName, report.ExporterUrl)
			}

			if report.ShouldBreak {
				breakResult, breakErr := watcher.circuitBreaker.BreakCircuit()
				if breakErr != nil {
					message := fmt.Sprintf("서킷브레이커가 아래와 같은 사유로 발동되어야 하지만 오류로 인해 실패하였습니다\n오류=%s\n사유=%s\n")
					watcher.mailSender.Send("[중요] 서킷브레이커 발동 실패", message)
				} else {
					circuitBreakerExecuted = true
					message := fmt.Sprintf("아래와 같은 사유로 서킷브레이커가 발동되었습니다.\n %s\n", report.Reason)
					message += fmt.Sprintf("%s\n", breakResult.Result)
					watcher.mailSender.Send("[중요] 서킷브레이커 발동 통보", message)
				}
			}
		}

		if circuitBreakerExecuted {
			break
		}
		time.Sleep(time.Duration(watcher.watchIntervalSec) * time.Second)
	}
}
