package job

import (
	"net"
	"strconv"

	"172.16.2.7/sre/monitoring-cms.git/internal/cmdb"
	"172.16.2.7/sre/monitoring-cms.git/internal/consul"
	"172.16.2.7/sre/monitoring-cms.git/internal/loggers"
	"github.com/spf13/viper"
)

type blackbox_exporterStruct struct {
	Instance           string            `json:"instance"`
	ObjectSummary      string            `json:"object_summary"`
	Labels             map[string]string `json:"labels"`
	MonitoringActivate string            `json:"monitoring_activate"`
}

// 把cmdb数据送到consul
func blackbox_exporter2ConsulJob() {
	// 查询cmdb数据
	result := []blackbox_exporterStruct{}

	_, err := cmdb.Query(
		"_type:ttpai_blackbox_exporter,monitoring_activate:True",
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

	loggers.DefaultLogger.Infof("从cmdb获取到%d条数据", len(result))

	// 获取consul客户端
	consulClient, err := consul.NewClient()
	if err != nil {
		loggers.DefaultLogger.Error("consul客户端创建失败:", err)
		return
	}

	queryCache := consul.Query(consulClient, viper.GetString("consul.job_name.blackbox_exporter"))

	// 解析cmdb数据，并往consul送
	for _, item := range result {

		host, port, err := net.SplitHostPort(item.Instance)
		if err != nil {
			// 从queryCache中移除已处理的服务
			delete(queryCache, viper.GetString("consul.job_name.blackbox_exporter")+"-"+item.Instance)
			loggers.DefaultLogger.Errorf("error splitting host and port from Instance %s: %v", item.Instance, err)
			continue
		}
		portInt, err := strconv.Atoi(port)
		if err != nil {
			// 从queryCache中移除已处理的服务
			delete(queryCache, viper.GetString("consul.job_name.blackbox_exporter")+"-"+item.Instance)
			loggers.DefaultLogger.Errorf("invalid port number '%s' in Instance '%s': %v", port, item.Instance, err)
			continue
		}

		// 是否开启监控自动发现
		tags := []string{}
		if item.MonitoringActivate == "True" {
			tags = []string{"activate"}
		}

		// 标签
		labels := map[string]string{
			"object_summary": getStringDefaultNull(item.ObjectSummary),
		}
		if item.Labels != nil {
			for k, v := range item.Labels {
				labels[k] = v
			}
		}

		consul.RegisterWithCache(
			consulClient,
			viper.GetString("consul.job_name.blackbox_exporter"),
			viper.GetString("consul.job_name.blackbox_exporter")+"-"+item.Instance,
			host,
			portInt,
			tags,
			labels,
			queryCache)
		// 从queryCache中移除已处理的服务
		delete(queryCache, viper.GetString("consul.job_name.blackbox_exporter")+"-"+item.Instance)
	}

	// 删除queryCache中所有没有被处理的服务
	for serviceID := range queryCache {
		consul.Deregister(consulClient, serviceID)
	}
	loggers.DefaultLogger.Info("删除了", len(queryCache), "个服务")
}
