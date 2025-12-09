package flag

import (
	"github.com/spf13/pflag"
)

var (
	Run = ""
)

// Init 初始化环境
func Init() {
	// --run 或 -r
	var run = pflag.StringP("run", "r", "", "单次执行，可选值：machine")

	//解析命令行参数
	pflag.Parse()

	Run = *run
}
