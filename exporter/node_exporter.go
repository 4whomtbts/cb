package exporter

import (
	"circuitbreaker_slave/util"
	"fmt"
	"github.com/go-resty/resty/v2"
	"github.com/google/logger"
	log "github.com/sirupsen/logrus"
	"regexp"
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
		return NewExporterReport(
			ne.exporterName, ne.url, false, false,
			"", fmt.Sprintf("Failed to fetch metric from %s !\n error was %s", ne.url, err.Error()))
	}
	body := string(resp.Body())
	lines := strings.Split(body, "\n")
	matchedAtLeastOnce := false
	r, _ := regexp.Compile("node_hwmon_temp_celsius{.+} [0-9]+")

	for _, line := range lines {
		if r.MatchString(line) {
			logger.Info("?????? ???????????? ?????????????????????")
			matchedAtLeastOnce = true
			toks := strings.Split(line, " ")
			cpuTemp, err :=  strconv.Atoi(toks[1])
			if err != nil {
				message := fmt.Sprintf("???????????? ?????? node exporter ???????????? ?????????????????????!metric=%s\n ??????=%s",
					line, err.Error())
				log.Errorf(err.Error(), message)
				return NewExporterReport(ne.exporterName, ne.url,false, false, "", message)
			}
			if cpuTemp > ne.maxCpuTemp {
				return NewExporterReport(ne.exporterName, ne.url,true, true,
					fmt.Sprintf(
						"[%s] ????????? CPU ????????? %d ??? ????????? ???????????? %d ?????? ????????????????????? ????????????????????? ???????????????!\n ????????? ??????:\n%s",
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
