# gogoPortScanTMP
* 运维人员:端口临时扫描工具,一款多功能TCP端口扫描与临时测试工具，集高效扫描、实时监听、多格式输出于一体，兼顾专业级性能与易用性。
* 程序名称:TCP端口扫描小工具
* 版本:v1.5-20250925@奔跑的老六**
## 使用示例:
	goPortScanTMP 127.0.0.1 22
	goPortScanTMP -ip 127.0.0.1 -p 22
	goPortScanTMP -net 192.168.88.0/24 -p 22,80 -a -v -o res.xlsx
	goPortScanTMP -l 8080,9000 -time 60 -v
## 参数说明:
	位置参数1    (IP地址,优先级低于 -ip)
	位置参数2    (端口,优先级低于 -p,默认22)
	-ip string   (IP地址,多个用逗号分隔)
	-net string  (网段,如:192.168.88.0/24)
	-p string    (端口/范围,如:22,1-100,默认22)
	-t duration  (超时时间,默认2秒)
	-n int       (重试次数,默认1)
	-u           (只输出关闭状态)
	-a           (只输出开放状态)
	-c int       (并发数,默认CPU核心数)
	-f string    (从文件读取IP,第一列有效)
	-o string    (输出文件,支持.json/.xlsx/.csv/.txt)
	-l string    (监听端口,如8080,9000)
	-time int    (监听持续时间(秒),默认30)
	-v           (详细日志模式)
