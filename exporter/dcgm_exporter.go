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

// DCGM_FI_DEV_GPU_TEMP{gpu="7",UUID="GPU-3f6d3448-1265-7b0d-8142-3fd8085e1072",device="nvidia7"} 26
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

	//if _, ok := config["url"]; !ok {
	//	panic("DcgmExporter's config should contain 'url'")
	//}
	//dcgmExporter.url = config["url"]
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
			"", fmt.Sprintf("메트릭 엔드포인트 %s 로부터 메트릭을 가져오는데 실패했습니다. error: %s", de.url, err.Error()))
	}
	body := string(resp.Body())
	lines := strings.Split(body, "\n")
	matchedAtLeastOnce := false
	for _, line := range lines {
		if strings.Contains(line, "DCGM_FI_DEV_GPU_TEMP") {
			matchedAtLeastOnce = true
			toks := strings.Split(line, " ")
			cpuTemp, err :=  strconv.Atoi(toks[1])
			if err != nil {
				message := fmt.Sprintf("올바르지 않은 node exporter 메트릭이 수집되었습니다!metric: %s\n 원인: %s",
					line, err.Error())
				log.Errorf(err.Error(), message)
				return NewExporterReport(de.exporterName, de.url,false, false, "", message)
			}
			if cpuTemp > de.maxGpuTemp {
				return NewExporterReport(de.exporterName, de.url,true, true,
					fmt.Sprintf(
						"[%s] 서버의 CPU 온도가 %d 로 설정된 최대온도 %d 도를 초과하였으므로 서킷브레이커를 발동합니다!\n 메트릭 전문: %s\n",
						de.url, cpuTemp, de.maxGpuTemp, lines), "")
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
