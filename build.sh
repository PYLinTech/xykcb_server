#!/bin/bash

cd "$(dirname "${BASH_SOURCE[0]}")"

echo "正在配置依赖..."
export GOPROXY=https://goproxy.cn,direct
go mod tidy

echo "正在编译..."
go build -o xykcb_server ./cmd/server

echo "编译完成：$(pwd)/xykcb_server"
