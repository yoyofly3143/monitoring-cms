package job

import (
	"strconv"

	"172.16.2.7/sre/monitoring-cms.git/internal/cmdb"
	"172.16.2.7/sre/monitoring-cms.git/internal/consul"
	"172.16.2.7/sre/monitoring-cms.git/internal/loggers"
	consulapi "github.com/hashicorp/consul/api"
)

type blackboxStruct struct {
	BlackboxEnv            string            `json:"blackbox_env"`
	BlackboxExporter       string            `json:"blackbox_exporter"`
	BlackboxExporterModule string            `json:"blackbox_exporter_module"`
	Instance               string            `json:"instance"`
	Labels                 map[string]string `json:"labels"`
	MonitoringActivate     string            `json:"monitoring_activate"`
	TtpaiBlackboxProbeId   int               `json:"ttpai_blackbox_probe_id"`
	MonitoringJob          string            `json:"monitoring_job"`
	ObjectSummary          string            `json:"object_summary"`
	Quality                int               `json:"quality"`
	ServiceName            string            `json:"service_name"`
	ServiceType            string            `json:"service_type"`
}

// 把cmdb黑盒探测数据送到consul
func blackbox2ConsulJob() {
	// 查询cmdb数据
	result := []blackboxStruct{}
	_, err := cmdb.Query(
		// 查询条件，模型
		"_type:ttpai_blackbox_probe",
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

	queryCacheMap := map[string]*map[string]*consulapi.AgentService{}

	// 解析cmdb数据，并往consul送
	for _, item := range result {

		// 是否开启监控自动发现
		tags := []string{}
		if item.MonitoringActivate == "True" {
			tags = []string{"activate"}
		}

		// 标签
		labels := map[string]string{
			"instance":          getStringDefaultNull(item.Instance),
			"blackbox_exporter": getStringDefaultNull(item.BlackboxExporter),
			"module":            getStringDefaultNull(item.BlackboxExporterModule),
			"env":               getStringDefaultNull(item.BlackboxEnv),
			"service_type":      getStringDefaultNull(item.ServiceType),
			"service_name":      getStringDefaultNull(item.ServiceName),
			"object_summary":    getStringDefaultNull(item.ObjectSummary),
			"quality":           strconv.Itoa(item.Quality),
		}
		if item.Labels != nil {
			for k, v := range item.Labels {
				labels[k] = v
			}
		}

		job := getStringDefaultNull(item.MonitoringJob)

		queryCachePtr, ok := queryCacheMap[job]
		if !ok {
			queryCache := consul.Query(consulClient, job)
			queryCacheMap[job] = &queryCache
			queryCachePtr = &queryCache
		}

		serviceID := job + "-" + strconv.Itoa(item.TtpaiBlackboxProbeId)

		// 注册服务并从缓存中移除
		consul.RegisterWithCache(
			consulClient,
			job,
			serviceID,
			"",
			0,
			tags,
			labels,
			*queryCachePtr)

		// 从queryCache中移除已处理的服务
		delete(*queryCachePtr, serviceID)
	}

	// 删除queryCache中所有没有被处理的服务
	for _, queryCachePtr := range queryCacheMap {
		for serviceID := range *queryCachePtr {
			consul.Deregister(consulClient, serviceID)
		}
		loggers.DefaultLogger.Info("删除了", len(*queryCachePtr), "个服务")
	}
}
