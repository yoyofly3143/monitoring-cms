package loggers

import (
	"io"
	"log"
	"os"

	"github.com/sirupsen/logrus"
)

const (
	logDir  = "./log"
	logPath = logDir + "/log-runtime.json"
)

var DefaultLogger = logrus.New()

func Init() {
	// 创建日志目录
	err := os.MkdirAll(logDir, os.ModePerm)
	if err != nil {
		log.Println("启动日志：创建日志目录失败：", err)
		os.Exit(3)
	}

	// 打开日志文件
	logFile, err := os.OpenFile(logPath, os.O_RDWR|os.O_CREATE|os.O_APPEND, 0644) //写入到文件中；此处使用追加写入，每一次都是追加写入。
	if err != nil {
		log.Println("启动日志：打开日志文件失败: ", err)
		os.Exit(3)
	}

	// 设定日志输出位置
	DefaultLogger.SetOutput(io.MultiWriter(os.Stderr, logFile))

	// 设定输出日志中是否要携带上文件名与行号
	DefaultLogger.SetReportCaller(false)

	// 设定日志等级
	DefaultLogger.SetLevel(logrus.InfoLevel)

	// 设定日志输出格式
	DefaultLogger.SetFormatter(
		&logrus.JSONFormatter{
			TimestampFormat: "2006-01-02 15:04:05.000 -0700 MST",
		},
	)

	DefaultLogger.Info("log模块初始化完成...")
}
