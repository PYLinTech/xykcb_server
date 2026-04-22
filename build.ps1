$env:GOPROXY="https://goproxy.cn,direct"
$env:GOSUMDB="off"

Write-Host "正在配置依赖..."
go mod tidy

Write-Host "正在编译..."
go build -o xykcb_server.exe .\cmd\server

if ($LASTEXITCODE -eq 0) {
    Write-Host "编译完成: $(Get-Location)\xykcb_server.exe"
} else {
    Write-Host "编译失败"
    exit 1
}
