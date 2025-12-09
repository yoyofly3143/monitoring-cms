package main

import (
	"os"

	"172.16.2.7/sre/monitoring-cms.git/internal/cmdb"
	"172.16.2.7/sre/monitoring-cms.git/internal/config"
	"172.16.2.7/sre/monitoring-cms.git/internal/flag"
	"172.16.2.7/sre/monitoring-cms.git/internal/job"
	"172.16.2.7/sre/monitoring-cms.git/internal/loggers"
)

func main() {
	// 初始化
	config.Init()
	flag.Init()
	loggers.Init()

	// 检查cmdb是否就绪
	cmdb.CheckClient()

	// 是否单次执行
	if flag.Run != "" {
		if flag.Run == "all" {
			job.DoAll()
		} else {
			err := job.Do(flag.Run)
			if err != nil {
				loggers.DefaultLogger.Errorln("错误的参数，没有这个任务，--run -r ：", flag.Run)
				os.Exit(3)
			}
		}
		return
	}

	// 启动定时任务
	job.Run()
}
