package consul

import (
	"fmt"
	"maps"
	"slices"
	"strings"

	"172.16.2.7/sre/monitoring-cms.git/internal/loggers"
	consulapi "github.com/hashicorp/consul/api"
	"github.com/spf13/viper"
)

func NewClient() (*consulapi.Client, error) {
	consulAddress := viper.GetString("consul.address")

	// 创建连接consul服务配置
	config := consulapi.DefaultConfig()
	config.Address = consulAddress
	client, err := consulapi.NewClient(config)
	if err != nil {
		loggers.DefaultLogger.Error("consul客户端创建失败:", err)
		return nil, err
	}
	return client, nil
}

func Register(client *consulapi.Client, serviceName string, serviceID string, address string, port int, tags []string, meta map[string]string) error {
	// 检查serviceID是否包含斜杠
	if strings.Contains(serviceID, "/") {
		loggers.DefaultLogger.Errorf("service ID '%s' contains a slash, which is not allowed", serviceID)
		return fmt.Errorf("service ID '%s' contains a slash, which is not allowed", serviceID)
	}

	for k, v := range viper.GetStringMapString("consul.labels") {
		meta[k] = v
	}

	// 创建注册到consul的服务到
	registration := new(consulapi.AgentServiceRegistration)
	registration.ID = serviceID     // 服务节点的名称
	registration.Name = serviceName // 服务名称
	registration.Port = port        // 服务端口
	registration.Tags = tags        // tag，可以为空
	registration.Address = address  // 服务 IP 要确保consul可以访问这个ip
	registration.Meta = meta

	// 注册服务到consul
	err := client.Agent().ServiceRegister(registration)
	if err != nil {
		loggers.DefaultLogger.Error(
			"consul服务注册失败:",
			" serviceName="+serviceName,
			" serviceID="+serviceID,
			" address="+address,
			" port="+fmt.Sprintf("%d", port),
			" tags="+fmt.Sprintf("%+v", tags),
			" meta="+fmt.Sprintf("%+v", meta),
			" err=", err)
		return err
	}
	loggers.DefaultLogger.Info(
		"consul服务注册成功：",
		" serviceName="+serviceName,
		" serviceID="+serviceID,
		" address="+address,
		" port="+fmt.Sprintf("%d", port),
		" tags="+fmt.Sprintf("%+v", tags),
		" meta="+fmt.Sprintf("%+v", meta),
	)
	return nil
}

// 根据服务组名称获取服务列表
func Query(client *consulapi.Client, serviceName string) map[string]*consulapi.AgentService {
	services, err := client.Agent().ServicesWithFilter("Service == \"" + serviceName + "\"")
	if err != nil {
		loggers.DefaultLogger.Error("查询consul失败：", err)
		return map[string]*consulapi.AgentService{}
	}
	return services
}

func Deregister(client *consulapi.Client, serviceID string) {
	err := client.Agent().ServiceDeregister(serviceID)
	if err != nil {
		loggers.DefaultLogger.Error("consul服务取消注册失败：", err)
	}
}

func RegisterWithCache(client *consulapi.Client, serviceName string, serviceID string, address string, port int, tags []string, meta map[string]string, queryCache map[string]*consulapi.AgentService) error {
	// 检查服务是否已经在缓存中
	cachedService, exists := queryCache[serviceID]
	if exists && cachedService.Service == serviceName && cachedService.ID == serviceID && cachedService.Address == address && cachedService.Port == port && slices.Equal(cachedService.Tags, tags) && maps.Equal(cachedService.Meta, meta) {
		// loggers.DefaultLogger.Info(
		// 	"consul服务不用再次注册：",
		// 	" serviceName="+serviceName,
		// 	" serviceID="+serviceID,
		// 	" address="+address,
		// 	" port="+fmt.Sprintf("%d", port),
		// 	" tags="+fmt.Sprintf("%+v", tags),
		// 	" meta="+fmt.Sprintf("%+v", meta),
		// )
		return nil
	} else {
		return Register(client, serviceName, serviceID, address, port, tags, meta)
	}
}
