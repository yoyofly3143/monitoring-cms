package metrics

import (
	"time"

	"172.16.2.7/sre/monitoring-cms.git/internal/loggers"
)

func Init() {

	// http.Handle("/metrics", promhttp.Handler())

}

func Do(name string, f func()) {
	start := time.Now()
	f()
	end := time.Now()
	d := end.Sub(start)
	loggers.DefaultLogger.Infoln(name, "执行时间：", d)
}
