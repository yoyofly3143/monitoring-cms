package job

import (
	"time"

	"172.16.2.7/sre/monitoring-cms.git/internal/cmdb"
	"172.16.2.7/sre/monitoring-cms.git/internal/consul"
	"172.16.2.7/sre/monitoring-cms.git/internal/loggers"
	"github.com/spf13/viper"
)

type machineStruct struct {
	CmdbAutoEnv         string            `json:"cmdb_auto_env"`
	CmdbAutoIpaddr      string            `json:"cmdb_auto_ipaddr"`
	CmdbAutoMachineType string            `json:"cmdb_auto_machine_type"`
	CmdbAutoOsType      string            `json:"cmdb_auto_os_type"`
	Labels              map[string]string `json:"labels"`
	MonitoringActivate  string            `json:"monitoring_activate"`
	ObjectSummary       string            `json:"object_summary"`
}

// 把cmdb机器数据送到consul
func machine2ConsulJob() {
	// 查询cmdb数据
	result := []machineStruct{}

	hour := time.Now().Add(-3 * time.Hour).Format("2006-01-02 15:04:05") // 三小时前
	_, err := cmdb.Query(
		// 查询条件，模型 环境 系统类型 更新时间
		"_type:ttpai_auto_machine_v2,cmdb_auto_env:prod,cmdb_auto_update_time:>"+hour,
		&result,
	)
	if err != nil {
		loggers.DefaultLogger.Error("查询ci模型数据时出错：", err)
		return
	}

	// 查到0条数据，不进行修改
	if len(result) == 0 {
		loggers.DefaultLogger.Warn("没有数据，不进行修改")
		return
	}

	loggers.DefaultLogger.Infof("从cmdb获取到%d台机器", len(result))

	// 获取consul客户端
	consulClient, err := consul.NewClient()
	if err != nil {
		loggers.DefaultLogger.Error("consul客户端创建失败:", err)
		return
	}

	queryCache := consul.Query(consulClient, viper.GetString("consul.job_name.machine"))

	// 解析cmdb数据，并往consul送
	for _, item := range result {
		// 是否开启监控自动发现
		tags := []string{}
		if item.MonitoringActivate == "True" {
			tags = []string{"activate"}
		}

		// 标签
		labels := map[string]string{
			"env":            item.CmdbAutoEnv,
			"machine_type":   item.CmdbAutoMachineType,
			"os_type":        item.CmdbAutoOsType,
			"object_summary": getStringDefaultNull(item.ObjectSummary),
		}
		if item.Labels != nil {
			for k, v := range item.Labels {
				labels[k] = v
			}
		}

		port := 0
		if item.CmdbAutoOsType == "linux" {
			port = 9100
		} else if item.CmdbAutoOsType == "windows" {
			port = 9182
		}

		consul.RegisterWithCache(
			consulClient,
			viper.GetString("consul.job_name.machine"),
			item.CmdbAutoIpaddr,
			item.CmdbAutoIpaddr,
			port,
			tags,
			labels,
			queryCache)

		// 从queryCache中移除已处理的服务
		delete(queryCache, item.CmdbAutoIpaddr)
	}

	// 删除queryCache中所有没有被处理的服务
	for serviceID := range queryCache {
		consul.Deregister(consulClient, serviceID)
	}
	loggers.DefaultLogger.Info("删除了", len(queryCache), "个服务")
}
