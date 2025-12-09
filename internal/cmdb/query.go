package cmdb

import (
	"encoding/json"
	"errors"
	"io"
	"net/http"

	"172.16.2.7/sre/monitoring-cms.git/internal/loggers"
)

type QuesyResp struct {
	Counter  map[string]int             `json:"counter"`  //	当前页按模型的分类统计
	Facet    map[string]json.RawMessage `json:"facet"`    //	返回的CI列表
	Numfound int                        `json:"numfound"` //	CI总数
	Page     int                        `json:"page"`     //	分页
	Result   json.RawMessage            `json:"result"`   //	返回的CI列表
	Total    int                        `json:"total"`    //	当前页的CI数
}

func Query(q string, result any) (*QuesyResp, error) {
	//定义CMDB的CI查询接口
	path := "/api/v0.1/ci/s"

	//	这里将默认的模型信息和变动的ci数据结合
	param := map[string]string{
		"sort":    "",     //	属性的排序，降序字段前面加负号-
		"page":    "1",    // 页数
		"count":   "9999", //	一页返回的CI数
		"ret_key": "name", //	返回字段类型,这里规定只能使用name
		"q":       q,
	}

	//拼接完整的CMDB连接串
	fullURL, err := Flurl(path, param)
	if err != nil {
		loggers.DefaultLogger.Error("CMDB客户端连接串配置错误", err)
		return nil, err
	}

	// 发送HTTP GET请求
	resp, err := http.Get(fullURL)
	if err != nil {
		loggers.DefaultLogger.Error("CMDB客户端连接检测请求发送失败：", err)
		return nil, err
	}
	defer resp.Body.Close()

	// 读取
	body, err := io.ReadAll(resp.Body)
	if err != nil {
		loggers.DefaultLogger.Error("CMDB读取查询结果失败：")
		return nil, err
	}

	if resp.StatusCode != 200 {
		loggers.DefaultLogger.Error("CMDB查询接口返回非200：", string(body))
		return nil, errors.New("CMDB查询接口返回非200")
	}

	queryResp := QuesyResp{}
	err = json.Unmarshal(body, &queryResp)
	if err != nil {
		loggers.DefaultLogger.Error("CMDB解析json失败：", err)
		return nil, err
	}

	err = json.Unmarshal(queryResp.Result, result)
	if err != nil {
		loggers.DefaultLogger.Error("CMDB解析json失败：", err)
		return nil, err
	}

	return &queryResp, nil
}
