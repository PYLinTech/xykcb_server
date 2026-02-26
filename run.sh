#!/bin/bash

cd "$(dirname "$0")"
clear

export GOPROXY=https://goproxy.cn,direct

for pid in $(ps -eo pid,comm | grep "xykcb_server$" | awk '{print $1}'); do
	echo "已停止正在运行的服务 (PID: $pid)"
	kill -9 $pid 2>/dev/null
done

echo "正在配置依赖..."
go mod tidy

echo "正在编译..."
go build -o xykcb_server ./cmd/server

echo "===== 服务日志 ====="
./xykcb_server
