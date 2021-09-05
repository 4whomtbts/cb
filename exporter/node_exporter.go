package exporter

import (
	"circuitbreaker_slave/util"
	"fmt"
	"github.com/go-resty/resty/v2"
	log "github.com/sirupsen/logrus"
	"strconv"
	"strings"
	"time"
)

type NodeExporter struct {
	Config map[string]string
	mailSender *util.MailSender
	restyClient *resty.Client
	exporterName string
	maxCpuTemp int
	url string
	mailIntervalSec int64
	mailEnabled bool
	lastMailSent int64
}

func NewNodeExporter(url string, config map[string]string) *NodeExporter {

	nodeExporter := &NodeExporter {}
	nodeExporter.url = url
	nodeExporter.mailEnabled = true

	if _, ok := config["name"]; !ok {
		nodeExporter.exporterName = "node_exporter" + strconv.FormatInt(time.Now().Unix(), 10)
	}

	//if _, ok := config["url"]; !ok {
	//	panic("NodeExporter's config should contain 'url'")
	//}
	//nodeExporter.url = config["url"]
	nodeExporter.restyClient = resty.New()

	if _, ok := config["maxCpuTemp"]; !ok {
		panic("NodeExporter's config should contain 'maxCpuTemp'")
	}
	maxCpuTemp, err := strconv.Atoi(config["maxCpuTemp"])
	if err != nil {
		panic("Failed to parse cpu temperature from NodeExporter metric")
	}
	nodeExporter.maxCpuTemp = maxCpuTemp

	if _, ok := config["mailIntervalSec"]; !ok {
		panic("NodeExporter's config should contain 'mailIntervalSec'")
	}
	nodeExporter.mailIntervalSec, err = strconv.ParseInt(config["mailIntervalSec"], 10, 64)
	if err != nil {
		nodeExporter.mailIntervalSec = 1800
		log.Warnf(
			"NodeExporter(%s)'s mailIntervalSec is set %d, since there is no provided mailIntervalSec option ")
	}
	return nodeExporter
}

func (ne *NodeExporter) EnableNotification(enable bool) {
	ne.mailEnabled = enable
}

func (ne *NodeExporter) sendMail(title string, content string, forceSend bool) {
	if !ne.mailEnabled && !forceSend {
		return
	}
	term := (time.Now().Unix() - ne.lastMailSent) / 1000
	if term < ne.mailIntervalSec && !forceSend {
		return
	}
	ne.lastMailSent = time.Now().Unix()
}

func (ne *NodeExporter) GetExporterReport() *ExporterReport {
	resp, err := ne.restyClient.R().EnableTrace().Get(ne.url)
	if err != nil {
		//content := fmt.Sprintf("NodeExporter(url=%s) 에서 메트릭을 받아오는데 실패했습니다.<br><br><h3>원인</h3><br>%s",
		//	ne.url, err.Error())
		//ne.sendMail("[서킷브레이커] 메트릭 수집 실패", content, false)
		return NewExporterReport(
			ne.exporterName, ne.url, false, false,
			"", fmt.Sprintf("Failed to fetch metric from %s !\n error was %s", ne.url, err.Error()))
	}
	body := string(resp.Body())
	lines := strings.Split(body, "\n")
	matchedAtLeastOnce := false
	for _, line := range lines {
		if strings.Contains(line, "node_hwmon_temp_celsius") {
			matchedAtLeastOnce = true
			toks := strings.Split(line, " ")
			cpuTemp, err :=  strconv.Atoi(toks[1])
			if err != nil {
				message := fmt.Sprintf("올바르지 않은 node exporter 메트릭이 수집되었습니다!metric=%s\n 원인=%s",
					line, err.Error())
				//ne.sendMail("메트릭 수집 실패", message, false)
				log.Errorf(err.Error(), message)
				return NewExporterReport(ne.exporterName, ne.url,false, false, "", message)
			}
			if cpuTemp > ne.maxCpuTemp {
				return NewExporterReport(ne.exporterName, ne.url,true, true,
					fmt.Sprintf(
						"[%s] 서버의 CPU 온도가 %d 로 설정된 최대온도 %d 도를 초과하였으므로 서킷브레이커를 발동합니다!\n 메트릭 전문:\n%s",
						ne.url, cpuTemp, ne.maxCpuTemp, lines), "")
			}
		}
	}
	if !matchedAtLeastOnce {
		return NewExporterReport(ne.exporterName, ne.url,true, true, "",
			fmt.Sprintf(
				"NodeExporter doesn't contain any cpu temperature metric(node_hwmon_temp_celsius)\n"))
	}
	return NewExporterReport(ne.exporterName, ne.url,false, true, "", "")
}