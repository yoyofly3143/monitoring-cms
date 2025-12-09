package job

import (
	"strconv"

	"172.16.2.7/sre/monitoring-cms.git/internal/cmdb"
	"172.16.2.7/sre/monitoring-cms.git/internal/consul"
	"172.16.2.7/sre/monitoring-cms.git/internal/loggers"
	"github.com/spf13/viper"
)

type mysqlStruct struct {
	ID                 int               `json:"ttpai_mysql_id"`
	DbIp               string            `json:"db_ip"`
	DbPort             int               `json:"db_port"`
	DbAddress          string            `json:"db_address"`
	TtpaiServiceSource string            `json:"ttpai_service_source"`
	MysqlExporter      string            `json:"mysql_exporter"`
	ObjectSummary      string            `json:"object_summary"`
	Labels             map[string]string `json:"labels"`
	MonitoringActivate string            `json:"monitoring_activate"`
}

// 把cmdb数据送到consul
func mysql2ConsulJob() {
	// 查询cmdb数据
	result := []mysqlStruct{}

	_, err := cmdb.Query(
		"_type:ttpai_mysql,~db_ip:null,~mysql_exporter:null,monitoring_activate:True",
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

	queryCache := consul.Query(consulClient, viper.GetString("consul.job_name.mysql"))

	// 解析cmdb数据，并往consul送
	for _, item := range result {

		// 是否开启监控自动发现
		tags := []string{}
		if item.MonitoringActivate == "True" {
			tags = []string{"activate"}
		}

		idc := ""
		if item.TtpaiServiceSource == "之家云" {
			idc = "prod-zhijia"
		} else if item.TtpaiServiceSource == "天天拍车" {
			idc = "prod-ttpai"
		} else {
			// 从queryCache中移除已处理的服务
			delete(queryCache, viper.GetString("consul.job_name.mysql")+"-"+strconv.Itoa(item.ID))
			loggers.DefaultLogger.Error("ttpai_mysql.ttpai_service_source 不是预期的值：", item.TtpaiServiceSource)
			continue
		}

		// 标签
		labels := map[string]string{
			"mysql_exporter": getStringDefaultNull(item.MysqlExporter),
			"object_summary": getStringDefaultNull(item.ObjectSummary),
			"idc":            idc,
		}
		if item.Labels != nil {
			for k, v := range item.Labels {
				labels[k] = v
			}
		}

		consul.RegisterWithCache(
			consulClient,
			viper.GetString("consul.job_name.mysql"),
			viper.GetString("consul.job_name.mysql")+"-"+strconv.Itoa(item.ID),
			item.DbIp,
			item.DbPort,
			tags,
			labels,
			queryCache)

		// 从queryCache中移除已处理的服务
		delete(queryCache, viper.GetString("consul.job_name.mysql")+"-"+strconv.Itoa(item.ID))
	}

	// 删除queryCache中所有没有被处理的服务
	for serviceID := range queryCache {
		consul.Deregister(consulClient, serviceID)
	}
	loggers.DefaultLogger.Info("删除了", len(queryCache), "个服务")
}
