# 小雨课程表服务端

小雨课程表服务端是一个 Go HTTP API 服务，用于获取学校课程、成绩和培养方案数据。当前主要实现 `hnit_a`，其他学校保留兼容存根。

## 版权信息

Copyright 2026 重庆沛雨霖科技有限公司 (PYLinTech)

Contact: PYLinTech@163.com

## 功能

| 功能 | 说明 |
|------|------|
| 课程数据 | 获取 HNIT 移动端课表数据，并输出统一课程结构 |
| 成绩查询 | 获取 HNIT 成绩和学期列表 |
| 培养方案 | 获取 HNIT 专业教学计划课程 |
| 学期配置 | 支持从远程第 2 周日期动态计算学期开始日 |
| 配置热更新 | 支持运行时更新服务器配置、学校配置和 404 页面 |
| Token 缓存 | 缓存登录 Token，失效后自动刷新并重试 |
| 速率限制 | 基于客户端 IP 限制请求频率 |
| CORS | 支持跨域配置 |

## 支持学校

| providerKey | 名称 | 状态 |
|-------------|------|------|
| hnit_a | 湖南工学院移动端 | 课程、成绩、培养方案 |
| hnit_b | 湖南工学院 PC 端 | 存根 |
| hynu | 衡阳师范学院 | 存根 |
| usc | 南华大学 | 存根 |

## 运行

### 环境

- Go 1.23 或更高版本

### 编译

```bash
go build -o xykcb_server ./cmd/server
```

Windows 可输出为：

```powershell
go build -o xykcb_server.exe ./cmd/server
```

### 启动

```bash
./xykcb_server
```

默认监听端口由 `assets/config.json` 中的 `server.port` 指定。

## API

所有 API 使用 `GET`，错误响应统一为：

```json
{
  "success": false,
  "desc_key": "001"
}
```

成功响应统一为：

```json
{
  "success": true,
  "data": {}
}
```

### 错误码

| desc_key | HTTP 状态码 | 说明 |
|----------|-------------|------|
| 001 | 400 | 缺少必要参数 |
| 002 | 404 | 不支持的学校 |
| 003 | 401 | 账户或密码错误 |
| 004 | 500 | 服务器内部错误 |
| 005 | 405 | 不支持的 HTTP 方法 |
| 006 | 500 | 获取数据失败 |
| 007 | 504 | 请求超时或功能未实现 |
| 008 | 401 | Token 已过期 |
| 009 | 429 | 频率超限 |

### 获取支持学校

```http
GET /get-support-school
```

响应示例：

```json
{
  "success": true,
  "data": [
    {"id": "1", "desc_key": "hnit_a"},
    {"id": "2", "desc_key": "hnit_b"},
    {"id": "3", "desc_key": "hynu"},
    {"id": "4", "desc_key": "usc"}
  ]
}
```

### 获取学校功能

```http
GET /get-support-function?school=hnit_a
```

### 获取课程数据

```http
GET /get-course-data?school=hnit_a&account=<account>&password=<password>
```

HNIT 课程输出结构：

```json
{
  "success": true,
  "data": {
    "2024-2025-2": {
      "semesterStart": "2025-02-17",
      "totalWeeks": 20,
      "timeSlots": [
        {"section": 1, "start": "08:30", "end": "09:15"}
      ],
      "mergeableSections": ["1-2", "3-4"],
      "courses": [
        {
          "id": "f2a3x",
          "rawId": "F0233264",
          "name": "计算机组成原理",
          "location": "1501",
          "teacher": "廖细生",
          "weeks": [2, 3, 4, 5, 7, 8, 9, 10, 11],
          "schedule": {"2": [7, 8]}
        }
      ]
    }
  }
}
```

课程字段说明：

| 字段 | 类型 | 说明 |
|------|------|------|
| id | string | 服务端处理后的课程唯一标识 |
| rawId | string | 学校接口返回的原始课程号 |
| name | string | 课程名称 |
| location | string | 上课地点，已移除括号备注 |
| teacher | string | 教师 |
| weeks | number[] | 上课周次 |
| schedule | object | 星期到节次数组的映射，`1` 为周一，`7` 为周日 |

### 获取成绩

```http
GET /get-course-grades?school=hnit_a&account=<account>&password=<password>&semester=<semester>
```

响应 `data` 包含：

| 字段 | 说明 |
|------|------|
| all-semester | 学期列表 |
| all-grades | 成绩数据 |

### 获取培养方案

```http
GET /get-guidance-teaching?school=hnit_a&account=<account>&password=<password>
```

## 配置

### `assets/config.json`

```json
{
  "server": {
    "port": "8080",
    "httpReadTimeout": 30,
    "httpWriteTimeout": 30,
    "rateLimit": 1000,
    "rateWindow": 60
  },
  "cors": {
    "allowAll": true,
    "allowedHosts": []
  }
}
```

### `assets/school_config.json`

`hnit_a` 使用 `semesterConfigFrom` 维护共享学期配置，服务端只动态计算每个学期的 `semesterStart`。

```json
{
  "hnit_a": {
    "semesterConfigTTL": 2592000,
    "semesterConfigFrom": [
      {
        "from": "2024-2025-2",
        "totalWeeks": 20,
        "mergeableSections": ["1-2", "3-4"],
        "timeSlots": [
          {"section": 1, "start": "08:30", "end": "09:15"}
        ]
      }
    ],
    "functions": []
  }
}
```

说明：

| 字段 | 说明 |
|------|------|
| semesterConfigTTL | 动态学期配置缓存时长，单位秒；未配置或小于等于 0 时默认 30 天 |
| semesterConfigFrom | 分段共享的学期配置列表 |
| from | 从该学期开始使用当前配置，比较时按去掉 `-` 后的数字排序 |
| totalWeeks | 学期总周数 |
| mergeableSections | 可合并节次 |
| timeSlots | 每节课起止时间 |
| functions | 学校功能入口配置 |

## HNIT 学期开始日计算

服务端会使用登录用户 Token 获取 `getXnxqList`，再对每个学期请求第 2 周课表：

```http
GET /student/curriculum?week=2&xnxq01id=<semester>
```

取 `data[0].date` 中 `xqid=1` 的 `mxrq`，回退 7 天得到该学期第一天。第 2 周用于计算日期，课程数据仍使用 `week=all` 获取。

如果某个学期开始日计算失败：

- 控制台输出失败日志。
- 不中断课程获取。
- 不刷新 TTL 缓存，后续用户请求会继续尝试直到成功。

## 项目结构

```text
.
├── assets/                    # 配置和静态页面
├── cmd/server/                # 程序入口
├── internal/app/              # 服务启动、路由和热更新
├── internal/cache/            # Token 缓存
├── internal/config/           # JSON 配置加载和文件监听
├── internal/errors/           # 错误码
├── internal/handler/          # HTTP handler 和中间件
├── internal/httpclient/       # 学校接口 HTTP 客户端
├── internal/model/            # 响应模型
└── internal/provider/         # 学校 provider 注册和实现
```

## 热更新

| 文件 | 行为 |
|------|------|
| assets/config.json | 重新加载配置并重启 HTTP 服务 |
| assets/school_config.json | 重新加载学校配置 |
| assets/404.html | 重新加载 404 页面 |

## 运行机制

- Token 缓存默认 5 分钟，过期自动清理。
- 学校接口返回非成功 `code` 时，服务端会使 Token 失效、重新登录并重试。
- HTTP 客户端默认 10 秒超时，遇到 5xx、连接失败、超时会重试。
- 速率限制基于客户端 IP，按配置窗口限制请求数。

## 许可证

本项目采用 Apache License, Version 2.0。详见 `LICENSE`。
