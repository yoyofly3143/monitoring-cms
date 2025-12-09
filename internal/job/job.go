package job

import (
	"errors"
	"strings"
	"time"

	"172.16.2.7/sre/monitoring-cms.git/internal/loggers"
	"172.16.2.7/sre/monitoring-cms.git/internal/metrics"
	"github.com/spf13/viper"
)

const (
	duration = 3 * time.Minute
)

// 这里放所有的定时任务
var jobs = map[string]func(){
	"online-app-service":          app_service2ConsulJob,
	"online-blackbox-app-service": blackbox_app_service2ConsulJob,
	"online-blackbox-exporter":    blackbox_exporter2ConsulJob,
	"online-blackbox":             blackbox2ConsulJob,
	"online-domain-ip":            domain_ip2ConsulJob,
	"online-machine":              machine2ConsulJob,
	"online-mysql":                mysql2ConsulJob,
}

// 单次执行所有定时任务
func DoAll() {
	// 把所有的定时任务执行一遍
	if viper.GetBool("job.env.online") {
		for k := range jobs {
			if strings.HasPrefix(k, "online-") {
				metrics.Do(k, jobs[k])
			}
		}
	}
	if viper.GetBool("job.env.offline") {
		for k := range jobs {
			if strings.HasPrefix(k, "offline-") {
				metrics.Do(k, jobs[k])
			}
		}
	}
}

// 单次执行定时任务
func Do(name string) error {
	job := jobs[name]
	if job == nil {
		return errors.New("没有这个任务：" + name)
	}
	metrics.Do(name, job)
	return nil
}

// 启动定时任务
func Run() {
	loggers.DefaultLogger.Info("定时任务启动成功")

	for {
		time.Sleep(duration)

		DoAll()
	}
}

func getStringDefaultNull(str string) string {
	if str == "" {
		return "null"
	}
	return str
}
