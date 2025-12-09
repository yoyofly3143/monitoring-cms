package job

import (
	"fmt"
	"time"

	"172.16.2.7/sre/monitoring-cms.git/internal/cmdb"
	"172.16.2.7/sre/monitoring-cms.git/internal/consul"
	"172.16.2.7/sre/monitoring-cms.git/internal/loggers"
	"github.com/spf13/viper"
)

type domain_ip_pubStruct struct {
	OID                      string `json:"oid"`
	CmdbAutoDnspodDomainfull string `json:"cmdb_auto_dnspod_domainfull"`
	CmdbAutoDnspodStatus     string `json:"cmdb_auto_dnspod_status"`
	CmdbAutoDnspodType       string `json:"cmdb_auto_dnspod_type"`
	CmdbAutoDnspodValue      string `json:"cmdb_auto_dnspod_value"`
	CmdbAutoUpdateTime       string `json:"cmdb_auto_update_time"`
	ObjectSummary            string `json:"object_summary"`
}

type domain_ip_intStruct struct {
	OID                     string `json:"oid"`
	CmdbAutoEnv             string `json:"cmdb_auto_env"`
	CmdbAutoPowerdnsName    string `json:"cmdb_auto_powerdns_name"`
	CmdbAutoPowerdnsType    string `json:"cmdb_auto_powerdns_type"`
	CmdbAutoPowerdnsContent string `json:"cmdb_auto_powerdns_content"`
	CmdbAutoUpdateTime      string `json:"cmdb_auto_update_time"`
	ObjectSummary           string `json:"object_summary"`
}

// 把cmdb黑盒探测数据送到consul
func domain_ip2ConsulJob() {
	// 查询cmdb数据
	resultPub := []domain_ip_pubStruct{}
	resultInt := []domain_ip_intStruct{}

	hour3 := time.Now().Add(-3 * time.Hour).Format("2006-01-02 15:04:05") // 三小时前
	_, err := cmdb.Query(
		// 查询条件
		"_type:ttpai_auto_dnspod,cmdb_auto_dnspod_status:True,cmdb_auto_dnspod_type:A,cmdb_auto_update_time:>"+hour3,
		&resultPub,
	)
	if err != nil {
		loggers.DefaultLogger.Error("查询ci模型数据时出错：", err)
		return
	}

	// 查到0条数据，不进行修改
	if len(resultPub) == 0 {
		loggers.DefaultLogger.Warn("没有数据，不进行修改")
		return
	}

	loggers.DefaultLogger.Infof("从cmdb获取到%d条公网域名数据", len(resultPub))

	hour6 := time.Now().Add(-6 * time.Hour).Format("2006-01-02 15:04:05")
	_, err = cmdb.Query(
		// 查询条件
		"_type:ttpai_auto_powerdns,cmdb_auto_powerdns_type:A,cmdb_auto_update_time:>"+hour6,
		&resultInt,
	)
	if err != nil {
		loggers.DefaultLogger.Error("查询ci模型数据时出错：", err)
		return
	}

	// 查到0条数据，不进行修改
	if len(resultInt) == 0 {
		loggers.DefaultLogger.Warn("没有数据，不进行修改")
		return
	}

	loggers.DefaultLogger.Infof("从cmdb获取到%d条内网域名数据", len(resultInt))

	// 获取consul客户端
	consulClient, err := consul.NewClient()
	if err != nil {
		loggers.DefaultLogger.Error("consul客户端创建失败:", err)
		return
	}

	queryCache := consul.Query(consulClient, viper.GetString("consul.job_name.domain_ip"))

	// 解析cmdb数据，并往consul送
	for _, item := range resultPub {

		// 标签
		labels := map[string]string{
			"instance":          getStringDefaultNull(item.CmdbAutoDnspodValue),
			"blackbox_exporter": "10.29.249.233:9115",
			"module":            "icmp",
			"sre_dns_env":       "public",
			"sre_domain":        getStringDefaultNull(item.CmdbAutoDnspodDomainfull),
			"object_summary":    getStringDefaultNull(item.ObjectSummary),
			"quality":           "9",
		}

		consul.RegisterWithCache(
			consulClient,
			viper.GetString("consul.job_name.domain_ip"),
			viper.GetString("consul.job_name.domain_ip")+fmt.Sprintf("-%s-%s-%s", "public", item.CmdbAutoDnspodDomainfull, item.CmdbAutoDnspodValue),
			"",
			0,
			[]string{},
			labels,
			queryCache)

		// 从queryCache中移除已处理的服务
		delete(queryCache, viper.GetString("consul.job_name.domain_ip")+fmt.Sprintf("-%s-%s-%s", "public", item.CmdbAutoDnspodDomainfull, item.CmdbAutoDnspodValue))
	}

	// 解析cmdb数据，并往consul送
	for _, item := range resultInt {

		// 标签
		labels := map[string]string{
			"instance":          getStringDefaultNull(item.CmdbAutoPowerdnsContent),
			"blackbox_exporter": "10.29.249.233:9115",
			"module":            "icmp",
			"sre_dns_env":       getStringDefaultNull(item.CmdbAutoEnv),
			"sre_domain":        getStringDefaultNull(item.CmdbAutoPowerdnsName),
			"object_summary":    getStringDefaultNull(item.ObjectSummary),
			"quality":           "9",
		}

		consul.RegisterWithCache(
			consulClient,
			viper.GetString("consul.job_name.domain_ip"),
			viper.GetString("consul.job_name.domain_ip")+fmt.Sprintf("-%s-%s-%s", item.CmdbAutoEnv, item.CmdbAutoPowerdnsName, item.CmdbAutoPowerdnsContent),
			"",
			0,
			[]string{"activate"},
			labels,
			queryCache)

		// 从queryCache中移除已处理的服务
		delete(queryCache, viper.GetString("consul.job_name.domain_ip")+fmt.Sprintf("-%s-%s-%s", item.CmdbAutoEnv, item.CmdbAutoPowerdnsName, item.CmdbAutoPowerdnsContent))
	}

	// 删除queryCache中所有没有被处理的服务
	for serviceID := range queryCache {
		consul.Deregister(consulClient, serviceID)
	}
	loggers.DefaultLogger.Info("删除了", len(queryCache), "个服务")
}
