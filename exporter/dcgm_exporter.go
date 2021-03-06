package exporter

import (
	"circuitbreaker_slave/util"
	"fmt"
	"github.com/go-resty/resty/v2"
	log "github.com/sirupsen/logrus"
	"regexp"
	"strconv"
	"strings"
	"time"
)

type DcgmExporter struct {
	Config map[string]string
	mailSender *util.MailSender
	restyClient *resty.Client
	exporterName string
	maxGpuTemp int
	url string
	mailIntervalSec int64
	mailEnabled bool
	lastMailSent int64
}

func NewDcgmExporter(url string, config map[string]string) *DcgmExporter {
	dcgmExporter := &DcgmExporter{}
	dcgmExporter.url = url
	dcgmExporter.mailEnabled = true
	if _, ok := config["name"]; !ok {
		dcgmExporter.exporterName = "dcgm_exporter" + strconv.FormatInt(time.Now().Unix(), 10)
	}

	dcgmExporter.restyClient = resty.New()

	if _, ok := config["maxGpuTemp"]; !ok {
		panic("DcgmExporter's config should contain 'maxGpuTemp'")
	}
	maxGpuTemp, err := strconv.Atoi(config["maxGpuTemp"])
	if err != nil {
		panic("Failed to parse gpu temperature from DcgmExporter metric")
	}
	dcgmExporter.maxGpuTemp = maxGpuTemp

	if _, ok := config["mailIntervalSec"]; !ok {
		panic("DcgmExporter's config should contain 'mailIntervalSec'")
	}
	dcgmExporter.mailIntervalSec, err = strconv.ParseInt(config["mailIntervalSec"], 10, 64)
	if err != nil {
		dcgmExporter.mailIntervalSec = 1800
		log.Warnf(
			"DcgmExporter(%s)'s mailIntervalSec is set %d, since there is no provided mailIntervalSec option ")
	}
	return dcgmExporter
}

func (de *DcgmExporter) EnableNotification(enable bool) {
	de.mailEnabled = enable
}

func (de *DcgmExporter) sendMail(title string, content string, forceSend bool) {
	if !de.mailEnabled && !forceSend {
		return
	}
	term := (time.Now().Unix() - de.lastMailSent) / 1000
	if term < de.mailIntervalSec && !forceSend {
		return
	}
	de.mailSender.Send(title, content)
	de.lastMailSent = time.Now().Unix()
}

func (de *DcgmExporter) GetExporterReport() *ExporterReport {
	resp, err := de.restyClient.R().EnableTrace().Get(de.url)
	if err != nil {
		return NewExporterReport(
			de.exporterName, de.url, false, false,
			"", fmt.Sprintf("????????? ??????????????? %s ????????? ???????????? ??????????????? ??????????????????. error: %s", de.url, err.Error()))
	}
	body := string(resp.Body())
	lines := strings.Split(body, "\n")
	matchedAtLeastOnce := false
	r, _ := regexp.Compile("DCGM_FI_DEV_GPU_TEMP{.+} [0-9]+")

	for _, line := range lines {
		if r.MatchString(line) {
			matchedAtLeastOnce = true
			toks := strings.Split(line, " ")
			gpuTemp, err := strconv.Atoi(toks[1])
			if err != nil {
				message := fmt.Sprintf("???????????? ?????? node exporter ???????????? ?????????????????????!metric: %s\n ??????: %s",
					line, err.Error())
				log.Errorf(err.Error(), message)
				return NewExporterReport(de.exporterName, de.url,false, false, "", message)
			}
			if gpuTemp > de.maxGpuTemp {
				return NewExporterReport(de.exporterName, de.url,true, true,
					fmt.Sprintf(
						"[%s] ????????? CPU ????????? %d ??? ????????? ???????????? %d ?????? ????????????????????? ????????????????????? ???????????????!\n ????????? ??????: %s\n",
						de.url, gpuTemp, de.maxGpuTemp, lines), "")
			}
		}
	}
	if !matchedAtLeastOnce {
		return NewExporterReport(de.exporterName, de.url,true, true, "",
			fmt.Sprintf(
				"DcgmExporter doesn't contain any cpu temperature metric(node_hwmon_temp_celsius)\n"))
	}
	return NewExporterReport(de.exporterName, de.url,false, true, "", "")
}
