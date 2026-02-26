#!/bin/bash

export GOPROXY=https://goproxy.cn,direct

echo "正在配置依赖..."
go mod tidy

echo "正在编译..."
go build -o xykcb_server ./cmd/server

echo "正在启动..."
./xykcb_server
