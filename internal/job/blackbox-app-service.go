package job

import (
	"net/url"
	"strconv"

	"172.16.2.7/sre/monitoring-cms.git/internal/cmdb"
	"172.16.2.7/sre/monitoring-cms.git/internal/consul"
	"172.16.2.7/sre/monitoring-cms.git/internal/loggers"
	consulapi "github.com/hashicorp/consul/api"
	"github.com/spf13/viper"
)

type ttpai_auto_app_dataStruct struct {
	CmdbAutoAppUniquekey string `json:"cmdb_auto_app_uniquekey"`
	CmdbAutoEnv          string `json:"cmdb_auto_env"`

	CmdbAutoTtpaiAppid  string `json:"cmdb_auto_ttpai_appid"`
	CmdbAutoAppName     string `json:"cmdb_auto_app_name"`
	CmdbAutoClusterName string `json:"cmdb_auto_cluster_name"`

	CmdbAutoTtpaiApplevel string `json:"cmdb_auto_ttpai_applevel"`
	CmdbAutoPriManager    string `json:"cmdb_auto_pri_manager"`

	ObjectSummary     string `json:"object_summary"`
	CmdbAutoAppNameCn string `json:"cmdb_auto_app_name_cn"`

	MonitoringInstance string            `json:"monitoring_instance"`
	Labels             map[string]string `json:"labels"`
	MonitoringActivate string            `json:"monitoring_activate"`
	// Quality            int               `json:"quality"`
}

func blackbox_app_service2ConsulJob() { // 查询cmdb数据
	result := []ttpai_auto_app_dataStruct{}
	_, err := cmdb.Query(
		"_type:ttpai_auto_app_data,cmdb_auto_cluster_name:(diana;ttpai;huawei-ttpai),monitoring_activate:True",
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

	queryCache := consul.Query(consulClient, viper.GetString("consul.job_name.blackbox_app_service"))

	hosts := ""
	hosts += "127.0.0.1   localhost localhost.localdomain localhost4 localhost4.localdomain4\n"
	hosts += "::1         localhost localhost.localdomain localhost6 localhost6.localdomain6\n"

	// 这里需要针对不同集群来添加不同的ingress入口监控
	for _, item := range result {
		if item.MonitoringInstance == "" {
			continue
		}
		// diana集群是GPU集群、和ttpai集群同属于廊坊集群
		if item.CmdbAutoClusterName == "diana" || item.CmdbAutoClusterName == "ttpai" {
			instance, err := url.Parse(item.MonitoringInstance)
			if err != nil {
				loggers.DefaultLogger.Error("MonitoringInstance解析错误:", err)
				continue
			}
			hostname := instance.Hostname()
			hosts += "10.28.3.149  " + hostname + "\n"

			// huawei-ttpai集群是华为云集群
		} else if item.CmdbAutoClusterName == "huawei-ttpai" {
			instance, err := url.Parse(item.MonitoringInstance)
			if err != nil {
				loggers.DefaultLogger.Error("MonitoringInstance解析错误:", err)
				continue
			}
			hostname := instance.Hostname()
			hosts += "10.41.3.102  " + hostname + "\n"
		}

	}

	// 将hosts内容存储到Consul
	_, err = consulClient.KV().Put(&consulapi.KVPair{
		Key:   "monitoring-cms/blackbox-app-service/hosts",
		Value: []byte(hosts),
	}, nil)
	if err != nil {
		loggers.DefaultLogger.Error("将hosts存储到Consul时出错:", err)
		return
	}

	loggers.DefaultLogger.Info("hosts内容已成功存储到Consul")

	// 解析cmdb数据，并往consul送
	for _, item := range result {
		// 是否开启监控自动发现
		tags := []string{}
		if item.MonitoringActivate == "True" {
			tags = []string{"activate"}
		}

		if item.MonitoringInstance == "" {
			tags = []string{}
		}

		objectSummary := item.CmdbAutoAppNameCn
		if item.ObjectSummary != "" {
			objectSummary = item.ObjectSummary
		}

		// 标签
		labels := map[string]string{
			"instance":          getStringDefaultNull(item.MonitoringInstance),
			"blackbox_exporter": "10.29.249.79:9115",
			"module":            "http_2xx",
			"env":               "prod",
			"service_type":      "app-service",
			"service_name":      getStringDefaultNull(item.CmdbAutoAppName),
			"appid":             getStringDefaultNull(item.CmdbAutoTtpaiAppid),
			"object_summary":    getStringDefaultNull(objectSummary),
			"quality":           strconv.Itoa(3),
			"sre_owner":         getStringDefaultNull(item.CmdbAutoPriManager),
			"sre_app_level":     getStringDefaultNull(item.CmdbAutoTtpaiApplevel),
		}
		if item.Labels != nil {
			for k, v := range item.Labels {
				labels[k] = v
			}
		}

		consul.RegisterWithCache(
			consulClient,
			viper.GetString("consul.job_name.blackbox_app_service"),
			viper.GetString("consul.job_name.blackbox_app_service")+"-"+item.CmdbAutoAppUniquekey,
			"",
			0,
			tags,
			labels,
			queryCache)
		// 从queryCache中移除已处理的服务
		delete(queryCache, viper.GetString("consul.job_name.blackbox_app_service")+"-"+item.CmdbAutoAppUniquekey)
	}

	// 删除queryCache中所有没有被处理的服务
	for serviceID := range queryCache {
		consul.Deregister(consulClient, serviceID)
	}
	loggers.DefaultLogger.Info("删除了", len(queryCache), "个服务")
}
