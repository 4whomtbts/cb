package exporter

type Exporter interface {
	GetExporterReport() *ExporterReport
	EnableNotification(enable bool)
}

type ExporterReport struct {
	ExporterName string
	ExporterUrl string
	ShouldBreak bool
	Success bool
	Reason string
	ErrLog string
}

func NewExporterReport(exporterName string, exporterUrl string, shouldBreak bool,
	success bool, reason string, errLog string) *ExporterReport {
	return &ExporterReport{
		ShouldBreak: shouldBreak,
		Success: success,
		Reason: reason,
		ExporterName: exporterName,
		ExporterUrl: exporterUrl,
		ErrLog: errLog,
	}
}