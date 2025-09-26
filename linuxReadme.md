# linux自行编译指导参考
## 1.尝试编译报错
	[root@laoliu PortScanTMP]# go build PortScanTMP
	package PortScanTMP is not in std (/usr/local/go/src/PortScanTMP)
	[root@laoliu PortScanTMP]# go build PortScanTMP.go
	PortScanTMP.go:18:5: no required module provides package github.com/xuri/excelize/v2:
	go.mod file not found in current directory or any parent directory; see 'go help modules'
	[root@laoliu PortScanTMP]# go get github.com/xuri/excelize/v2
	go: go.mod file not found in current directory or any parent directory.
	'go get' is no longer supported outside a module.
	To build and install a command, use 'go install' with a version,
	like 'go install example.com/cmd@latest'
	For more information, see https://golang.org/doc/go-get-install-deprecation
	or run 'go help get' or 'go help install'.
## 3.先初始化模块（必须步骤）
	[root@laoliu PortScanTMP]# go mod init PortScanTMP
	go: creating new go.mod: module PortScanTMP
	go: to add module requirements and sums:
		go mod tidy
	[root@laoliu PortScanTMP]# ls
	go.mod  PortScanTMP.go
	[root@laoliu PortScanTMP]# more go.mod
	module PortScanTMP

	go 1.24.7
	[root@laoliu PortScanTMP]# go run PortScanTMP.go
	PortScanTMP.go:18:5: no required module provides package github.com/xuri/excelize/v2; to add it:
		go get github.com/xuri/excelize/v2
  ## 2.增加代理
	[root@laoliu PortScanTMP]# goProxy
	godaili:https://goproxy.cn,direct
	[root@jiankong-prod-mgmt-20250210 PortScanTMP]# cat /bin/goProxy
	godaili="https://goproxy.cn,direct"
	go env -w GO111MODULE=on
	go env -w GOPROXY=$godaili
	echo godaili:$godaili
	[root@laoliu PortScanTMP]#
  ## 3.下载依赖包
	[root@laoliu PortScanTMP]# go get github.com/xuri/excelize/v2
	go: added github.com/richardlehane/mscfb v1.0.4
	go: added github.com/richardlehane/msoleps v1.0.4
	go: added github.com/tiendc/go-deepcopy v1.6.0
	go: added github.com/xuri/efp v0.0.1
	go: added github.com/xuri/excelize/v2 v2.9.1
	go: added github.com/xuri/nfp v0.0.1
	go: added golang.org/x/crypto v0.38.0
	go: added golang.org/x/net v0.40.0
	go: added golang.org/x/text v0.25.0
  ## 4.编译
	[root@laoliu PortScanTMP]# go build PortScanTMP.go
  ## 5.压缩(提前准备upx)
	[root@laoliu PortScanTMP]# upx -9 PortScanTMP
						   Ultimate Packer for eXecutables
							  Copyright (C) 1996 - 2024
	UPX 4.2.2       Markus Oberhumer, Laszlo Molnar & John Reiser    Jan 3rd 2024

			File size         Ratio      Format      Name
	   --------------------   ------   -----------   -----------
	   9858320 ->   5071440   51.44%   linux/amd64   PortScanTMP

	Packed 1 file.
	[root@laoliu PortScanTMP]#
	[root@laoliu PortScanTMP]# du -sh *
	4.0K	go.mod
	4.0K	go.sum
	4.9M	PortScanTMP
	28K	PortScanTMP.go
  ## 6.使用说明
	[root@laoliu PortScanTMP]# ./PortScanTMP
	===========================================================
	程序名称:TCP端口扫描小工具  版本:v1.5-20250925@奔跑的老六
	使用示例:
	  PortScanTMP 127.0.0.1 22
	  PortScanTMP -ip 127.0.0.1 -p 22
	  PortScanTMP -net 192.168.0/24 -p 22,80 -a -v -o res.xlsx
	  PortScanTMP -l 8080,9000 -time 60 -v

	参数说明:
	  位置参数1    (IP地址,优先级低于 -ip)
	  位置参数2    (端口,优先级低于 -p,默认22)
	  -ip string   (IP地址,多个用逗号分隔)
	  -net string  (网段,如:192.168.0/24)
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
	===========================================================
	[root@laoliu PortScanTMP]#
  	[root@laoliu PortScanTMP]# ./PortScanTMP 192.168.88.88
	2025-09-26_02:19:48	192.168.88.88	22	open
	[root@laoliu PortScanTMP]# ./PortScanTMP 192.168.88.88 88
	2025-09-26_02:19:51	192.168.88.88	88	close
	[root@laoliu PortScanTMP]# ./PortScanTMP 192.168.88.88 88 -o a.json
	[root@laoliu PortScanTMP]# cat a.json
	[
	  {
		"time": "2025-09-26T02:20:00.815927922Z",
		"ip": "192.168.88.88",
		"port": "88",
		"status": "close"
	  }
	]
	[root@laoliu PortScanTMP]#
  ## 6.使用说明1 默认扫描22端口 -o输出到文件,支持.json/.xlsx/.csv/.txt,必须指定文件名
	[root@laoliu PortScanTMP]# ./PortScanTMP 192.168.88.88 -o a.txt
	[root@laoliu PortScanTMP]# cat a.txt
	时间	IP	端口	状态
	2025-09-26 02:21:13	192.168.88.88	22	open
	[root@laoliu PortScanTMP]# ./PortScanTMP 192.168.88.88 -o a.csv
	[root@laoliu PortScanTMP]# cat a.csv
	时间,IP,端口,状态
	2025-09-26 02:21:26,192.168.88.88,22,open
	[root@laoliu PortScanTMP]#
  ## 7.使用说明2:启动临时端口方便网络测试,支持多个端口逗号分割
	[root@laoliu PortScanTMP]# ./PortScanTMP -l 88,99,22
	2025/09/26 10:25:50 监听模式:详细输出已开启(时间≤30秒或指定了-v)
	2025/09/26 10:25:50 启动本地监听,持续 30 秒
	2025/09/26 10:25:50 开始监听 0.0.0.0:99,持续 30 秒
	2025/09/26 10:25:50 开始监听 0.0.0.0:88,持续 30 秒
	2025/09/26 10:25:50 错误:端口 22 已被占用,无法监听
	2025/09/26 10:25:51 端口 88 将在 29 秒后关闭
	2025/09/26 10:25:51 端口 99 将在 29 秒后关闭
	2025/09/26 10:25:52 端口 88 将在 28 秒后关闭
	2025/09/26 10:25:52 端口 99 将在 28 秒后关闭
	2025/09/26 10:25:53 端口 88 将在 27 秒后关闭
	2025/09/26 10:25:53 端口 99 将在 27 秒后关闭
	2025/09/26 10:25:54 端口 88 将在 26 秒后关闭
	2025/09/26 10:25:54 端口 99 将在 26 秒后关闭
	2025/09/26 10:25:55 端口 88 将在 25 秒后关闭
	2025/09/26 10:25:55 端口 99 将在 25 秒后关闭
	[root@laoliu PortScanTMP]# ./PortScanTMP -l 88,99,22 -time 10
	2025/09/26 10:28:34 监听模式:详细输出已开启(时间≤30秒或指定了-v)
	2025/09/26 10:28:34 启动本地监听,持续 10 秒
	2025/09/26 10:28:34 开始监听 0.0.0.0:99,持续 10 秒
	2025/09/26 10:28:34 错误:端口 22 已被占用,无法监听
	2025/09/26 10:28:34 开始监听 0.0.0.0:88,持续 10 秒
	2025/09/26 10:28:34 收到来自 192.168.88.89:53098 的连接
	2025/09/26 10:28:35 端口 88 将在 9 秒后关闭
	2025/09/26 10:28:35 端口 99 将在 9 秒后关闭
	2025/09/26 10:28:36 端口 88 将在 8 秒后关闭
	2025/09/26 10:28:36 端口 99 将在 8 秒后关闭
	2025/09/26 10:28:37 端口 88 将在 7 秒后关闭
	2025/09/26 10:28:37 端口 99 将在 7 秒后关闭
	2025/09/26 10:28:38 端口 88 将在 6 秒后关闭
	2025/09/26 10:28:38 端口 99 将在 6 秒后关闭
	2025/09/26 10:28:39 端口 88 将在 5 秒后关闭
	2025/09/26 10:28:39 端口 99 将在 5 秒后关闭
	2025/09/26 10:28:40 端口 99 将在 4 秒后关闭
	2025/09/26 10:28:40 端口 88 将在 4 秒后关闭
	2025/09/26 10:28:41 端口 88 将在 3 秒后关闭
	2025/09/26 10:28:41 端口 99 将在 3 秒后关闭
	2025/09/26 10:28:42 端口 99 将在 2 秒后关闭
	2025/09/26 10:28:42 端口 88 将在 2 秒后关闭
	2025/09/26 10:28:43 端口 88 将在 1 秒后关闭
	2025/09/26 10:28:43 端口 99 将在 1 秒后关闭
	2025/09/26 10:28:44 监听结束 0.0.0.0:88
	2025/09/26 10:28:44 警告:接受连接失败 0.0.0.0:88:accept tcp [::]:88: use of closed network connection
	2025/09/26 10:28:44 监听结束 0.0.0.0:99
	2025/09/26 10:28:44 警告:接受连接失败 0.0.0.0:99:accept tcp [::]:99: use of closed network connection
	2025/09/26 10:28:44 所有监听已结束
	[root@laoliu PortScanTMP]#
