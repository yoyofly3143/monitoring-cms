package config

import (
	"log"
	"os"

	"github.com/spf13/viper"
)

const (
	// 配置文件地址
	defaultConfigPath = "./conf/conf.yaml"
)

func Init() {
	//设置加载的配置文件
	viper.SetConfigFile(defaultConfigPath)

	//读取配置文件（读取上面指定的配置文件）
	err := viper.ReadInConfig()
	if err != nil {
		log.Println("启动日志："+defaultConfigPath+"配置文件读取失败：", err)
		os.Exit(3)
	} else {
		log.Println("启动日志：" + defaultConfigPath + "配置文件读取完成")
	}
}
