#!/bin/bash

cd "$(dirname "${BASH_SOURCE[0]}")"

export GOPROXY=https://goproxy.cn,direct
export GOSUMDB=off

echo "正在配置依赖..."
go mod tidy

if [ $? -ne 0 ]; then
    echo "配置依赖失败"
    exit 1
fi

echo "正在编译..."
go build -o xykcb_server ./cmd/server

if [ $? -eq 0 ]; then
    echo "编译完成: $(pwd)/xykcb_server"
else
    echo "编译失败"
    exit 1
fi
