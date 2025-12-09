package job

import (
	"172.16.2.7/sre/monitoring-cms.git/internal/cmdb"
	"172.16.2.7/sre/monitoring-cms.git/internal/consul"
	"172.16.2.7/sre/monitoring-cms.git/internal/loggers"
	"github.com/spf13/viper"
)

type ttpai_auto_k8s_instanceStruct struct {
	Cmdb_auto_instance_name string `json:"cmdb_auto_instance_name"`
	Cmdb_auto_instance_ip   string `json:"cmdb_auto_instance_ip"`
	Cmdb_auto_app_name      string `json:"cmdb_auto_app_name"`
	Cmdb_auto_env           string `json:"cmdb_auto_env"`
}

func app_service2ConsulJob() { // 查询cmdb数据
	result := []ttpai_auto_k8s_instanceStruct{}
	_, err := cmdb.Query(
		"_type:ttpai_auto_k8s_instance,cmdb_auto_env:prod*",
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

	queryCache := consul.Query(consulClient, viper.GetString("consul.job_name.app_service"))

	// 解析cmdb数据，并往consul送
	for _, item := range result {
		tags := []string{"activate"}

		labels := map[string]string{
			"zj_env":       item.Cmdb_auto_env,
			"service_name": item.Cmdb_auto_app_name,
			"pod":          item.Cmdb_auto_instance_name,
			"pod_ip":       item.Cmdb_auto_instance_ip,
		}

		consul.RegisterWithCache(
			consulClient,
			viper.GetString("consul.job_name.app_service"),
			viper.GetString("consul.job_name.app_service")+"-"+item.Cmdb_auto_instance_name,
			item.Cmdb_auto_instance_ip,
			8080,
			tags,
			labels,
			queryCache)

		// 从queryCache中移除已处理的服务
		delete(queryCache, viper.GetString("consul.job_name.app_service")+"-"+item.Cmdb_auto_instance_name)
	}

	// 删除queryCache中所有没有被处理的服务
	for serviceID := range queryCache {
		consul.Deregister(consulClient, serviceID)
	}
	loggers.DefaultLogger.Info("删除了", len(queryCache), "个服务")
}
