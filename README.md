## 运行
```
go run ./cmd/monitoring-cms.go
```
单次执行
```
go run ./cmd/monitoring-cms.go -r all
```


monitor-cms本身的各种监控数据是从cmdb中采集的
- internal/job：在这个里面定义了从cmdb中取到的不同的数据源，然后注册到consul

应用的大致链路描述如下
1. monitor-cms跟cmdb进行通信，获取到数据
2. monitor-cms将数据添加标签注册到consul中
3. prometheus从consul中获取数据元信息，将其加为监控项，然后由blackbox_exporter进行监控

特殊说明
1. 针对应用的监控，会用到consul的key/value的功能
2. monitor-cms从cmdb中获取到应用数据后，会根据应用所在集群，确定其所在机房
   1. monitoring-cms代码文件：internal/job/blackbox-app-service.go
   2. 代码行数：60行～92行
3. 然后域名+ingress数据注册到consul的key/value中
   1. http://10.29.249.234:8500/ui/prometheus-consul/kv/monitoring-cms/blackbox-app-service/hosts/edit
4. 再由10.29.249.79机器中的 consul-template 应用将 consul中key/value数据写入到10.29.249.79的/etc/hosts文件中
   1. https://github.com/hashicorp/consul-template
5. 最后，blackbox_exporter通过hosts定义的域名和ip的关系，去到不同入口对应用进行监控