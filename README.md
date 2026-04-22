# 小雨课程表服务端

小雨课程表服务端是一个用 Go 语言开发的高并发课程数据获取 API 服务，支持多学校登录、配置文件热更新、HTTP 连接池管理、Token 缓存自动清理等功能。

## 版权信息

Copyright 2026 重庆沛雨霖科技有限公司 (PYLinTech)
Contact: PYLinTech@163.com

## 功能特性

| 功能 | 说明 |
|------|------|
| 多学校支持 | 支持湖南工学院、衡阳师范学院、南华大学等多个学校的登录和课程数据获取 |
| 配置文件热更新 | 支持在运行期间动态更新服务器配置、学校配置和404页面，无需重启服务 |
| 服务器优雅关闭 | 支持通过信号（SIGINT/SIGTERM）优雅关闭服务器，确保正在处理的请求能够完成 |
| HTTP连接池管理 | 内置 HTTP 客户端使用连接池，复用 TCP 连接，减少连接建立开销 |
| Token缓存自动清理 | 内置 Token 缓存模块，每分钟自动清理过期的 Token，防止内存泄漏 |
| 请求速率限制 | 基于 IP 的请求速率限制功能，防止滥用和保护服务稳定性 |
| 请求ID追踪 | 为每个请求生成唯一 ID，添加到响应头 X-Request-ID |
| 统一错误处理 | 规范化的错误码体系，便于前端统一处理错误场景 |
| 运行时指标监控 | 每分钟输出访问量、用户数、Token缓存命中率等运行指标 |

## 支持学校

| providerKey | 中文名称 | 英文名称 | 功能支持 |
|-------------|----------|----------|----------|
| hnit_a | 湖南工学院（移动端） | Hunan Institute Of Technology (Mobile) | 课程数据、成绩、专业计划课程 |
| hnit_b | 湖南工学院（PC端） | Hunan Institute Of Technology (PC) | 存根实现 |
| hynu | 衡阳师范学院 | Hengyang Normal University | 存根实现 |
| usc | 南华大学 | University of South China | 存根实现 |

## 系统要求

- Go 1.21 或更高版本
- 支持 Windows/Linux/macOS

## 快速开始

### 编译

```bash
# Windows PowerShell
.\build.ps1

# Linux/macOS
./build.sh
```

编译完成后会生成 `xykcb_server.exe`（Windows）或 `xykcb_server`（Linux/macOS）。

### 运行

```bash
# Linux/macOS
./xykcb_server

# Windows
.\xykcb_server
```

服务器将在 http://localhost:8080 启动。

### 停止服务

```bash
# 按 Ctrl+C 发送 SIGINT 信号
# 或使用 kill 命令发送 SIGTERM 信号
kill -15 <pid>
```

## API 接口

所有 API 支持 CORS，可通过配置允许跨域请求。

### 通用响应格式

成功响应：
```json
{
  "success": true,
  "data": { ... }
}
```

错误响应：
```json
{
  "success": false,
  "desc_key": "003"
}
```

### 错误码对照表

| desc_key | HTTP状态码 | 含义 | 处理建议 |
|----------|------------|------|----------|
| 001 | 400 | 缺少必要参数 | 检查请求参数是否完整 |
| 002 | 404 | 不支持的学校 | 确认 school 参数正确 |
| 003 | 401 | 账户或密码错误 | 检查账号密码是否正确 |
| 004 | 500 | 登录失败 | 学校服务可能异常，稍后重试 |
| 005 | 405 | 不支持的HTTP方法 | 只支持 GET 和 OPTIONS |
| 006 | 500 | 获取数据失败 | 学校服务可能异常，稍后重试 |
| 007 | 504 | 请求超时 | 学校服务响应慢，稍后重试 |
| 008 | 401 | Token已过期 | 系统自动刷新，稍后重试 |
| 009 | 429 | 频率超限 | 降低请求频率，等待后重试 |

### 获取支持学校列表

获取当前服务器支持的所有学校列表。

**请求**
```
GET /get-support-school
```

**响应示例**
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

### 获取学校功能列表

获取指定学校支持的功能列表。

**请求**
```
GET /get-support-function?school=<providerKey>
```

**参数**
| 参数名 | 类型 | 必填 | 说明 |
|--------|------|------|------|
| school | string | 是 | 学校 providerKey（如 hnit_a, hnit_b, hynu, usc） |

**响应示例**
```json
{
  "success": true,
  "data": [
    { "id": "1", "url": "/functions/hnit_a/grades.html", "zh-cn": "课程成绩", "en": "Course Grades" }
  ]
}
```

### 获取课程数据

获取指定学校的课程表数据。该接口会并发获取所有学期的课程数据。

**请求**
```
GET /get-course-data?school=<providerKey>&account=<account>&password=<password>
```

**参数**
| 参数名 | 类型 | 必填 | 说明 |
|--------|------|------|------|
| school | string | 是 | 学校 providerKey（如 hnit_a） |
| account | string | 是 | 账号（学号） |
| password | string | 是 | 密码 |

**响应示例**（湖南工学院移动端）
```json
{
  "success": true,
  "data": {
    "2025-2026-2": {
      "semesterStart": "2026-03-02",
      "totalWeeks": 20,
      "timeSlots": [
        {"section": 1, "start": "08:30", "end": "09:15"},
        {"section": 2, "start": "09:20", "end": "10:05"},
        {"section": 3, "start": "10:25", "end": "11:10"},
        {"section": 4, "start": "11:15", "end": "12:00"},
        {"section": 5, "start": "14:00", "end": "14:45"},
        {"section": 6, "start": "14:50", "end": "15:35"},
        {"section": 7, "start": "15:55", "end": "16:40"},
        {"section": 8, "start": "16:45", "end": "17:30"},
        {"section": 9, "start": "19:00", "end": "19:45"},
        {"section": 10, "start": "19:50", "end": "20:35"}
      ],
      "mergeableSections": ["1-2", "3-4", "5-6", "7-8", "9-10"],
      "courses": [
        {
          "id": "A001",
          "name": "高等数学",
          "location": "教学楼A101",
          "teacher": "张三",
          "weeks": [1,2,3,4,5,6,7,8,9,10,11,12,13,14,15,16,17,18],
          "schedule": {
            "1": [1, 2],
            "3": [3, 4]
          }
        }
      ]
    }
  }
}
```

**课程数据结构说明**
| 字段 | 类型 | 说明 |
|------|------|------|
| id | string | 课程代码 |
| name | string | 课程名称 |
| location | string | 上课地点（已清理括号内的备注信息） |
| teacher | string | 授课教师 |
| weeks | array | 上课周次，如 [1,3,5] 表示第1、3、5周上课 |
| schedule | object | 上课时间，键为星期几，值为节次数组 |

**schedule 字段说明**

schedule 是一个对象，键为星期几（1-7 表示周一到周日），值为该星期上课的节次数组。

例如 `"schedule": {"1": [1, 2], "3": [3, 4]}` 表示：
- 周一：第1-2节上课
- 周三：第3-4节上课

### 获取成绩数据

获取指定学校的成绩数据。

**请求**
```
GET /get-course-grades?school=<providerKey>&account=<account>&password=<password>&semester=<semester>
```

**参数**
| 参数名 | 类型 | 必填 | 说明 |
|--------|------|------|------|
| school | string | 是 | 学校 providerKey |
| account | string | 是 | 账号 |
| password | string | 是 | 密码 |
| semester | string | 是 | 学期标识（如 2025-2026-2） |

**响应示例**
```json
{
  "success": true,
  "data": {
    "all-semester": [...],
    "all-grades": [...]
  }
}
```

### 获取专业计划课程

获取指定学校的专业教学计划课程数据（人才培养方案中的课程列表）。

**请求**
```
GET /get-guidance-teaching?school=<providerKey>&account=<account>&password=<password>
```

**参数**
| 参数名 | 类型 | 必填 | 说明 |
|--------|------|------|------|
| school | string | 是 | 学校 providerKey |
| account | string | 是 | 账号 |
| password | string | 是 | 密码 |

**响应示例**
```json
{
  "success": true,
  "data": [
    {
      "courseName": "高等数学",
      "courseType": "公共基础课",
      "credit": 4,
      "hours": 64,
      "semester": "第一学年第一学期"
    }
  ]
}
```

**响应字段说明**

| 字段 | 类型 | 说明 |
|------|------|------|
| courseName | string | 课程名称 |
| courseType | string | 课程类型（如公共基础课、专业课、选修课） |
| credit | number | 学分 |
| hours | number | 学时 |
| semester | string | 开设学期 |

## 配置

### 目录结构

```
.
├── assets/                    # 静态资源配置目录
│   ├── config.json           # 服务器配置
│   ├── school_config.json    # 学校配置
│   └── 404.html              # 404错误页面
├── cmd/server/               # 应用入口
│   └── main.go               # 主程序入口
├── internal/                 # 内部包
│   ├── cache/                # Token缓存
│   ├── config/               # 配置管理
│   ├── errors/               # 错误定义
│   ├── handler/              # HTTP处理器和中间件
│   ├── httpclient/           # HTTP客户端封装
│   ├── metrics/              # 运行指标统计
│   ├── model/                # 数据模型
│   └── provider/             # 学校提供者
│       └── schools/          # 各学校具体实现
├── .gitignore                # Git忽略配置
├── build.ps1                 # Windows编译脚本
├── build.sh                  # Linux/macOS编译脚本
├── go.mod                    # Go模块定义
├── go.sum                    # 依赖校验
├── LICENSE                   # 许可证
├── NOTICE                    # 注意事项
└── README.md                 # 本文档
```

### 服务器配置 (assets/config.json)

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

**配置项说明**

| 字段 | 类型 | 默认值 | 说明 |
|------|------|--------|------|
| server.port | string | "8080" | 服务器监听端口 |
| server.httpReadTimeout | int | 30 | HTTP 请求读取超时时间（秒） |
| server.httpWriteTimeout | int | 30 | HTTP 响应写入超时时间（秒） |
| server.rateLimit | int | 1000 | 速率限制：时间窗口内允许的最大请求数 |
| server.rateWindow | int | 60 | 速率限制时间窗口（秒） |
| cors.allowAll | bool | true | 是否允许所有域名跨域访问 |
| cors.allowedHosts | array | [] | 允许跨域的域名列表，当 allowAll 为 false 时生效 |

### 学校配置 (assets/school_config.json)

```json
{
  "hnit_a": {
    "semesters": {
      "2025-2026-1": {
        "semesterStart": "2025-09-01",
        "totalWeeks": 20,
        "timeSlots": [
          {"section": 1, "start": "08:00", "end": "08:45"}
        ],
        "mergeableSections": ["1-2", "3-4"]
      }
    },
    "functions": [
      { "id": "1", "url": "/functions/hnit_a/grades.html", "zh-cn": "课程成绩", "en": "Course Grades" }
    ]
  }
}
```

**配置项说明**

| 字段 | 类型 | 说明 |
|------|------|------|
| semesters | object | 学期配置，键为学期标识 |
| semesterStart | string | 学期开始日期（YYYY-MM-DD） |
| totalWeeks | int | 学期总周数 |
| timeSlots | array | 每天的课程时间段配置 |
| timeSlots[].section | int | 节次编号（1-10） |
| timeSlots[].start | string | 开始时间（HH:MM） |
| timeSlots[].end | string | 结束时间（HH:MM） |
| mergeableSections | array | 可合并的课程节次，如 "1-2" 表示第1-2节可以合并显示 |
| functions | array | 学校功能列表 |

### 404 页面 (assets/404.html)

用户访问不存在的路径时返回的 HTML 页面内容。

## 中间件

服务使用链式中间件架构，请求处理顺序为：

```
RequestID → Logging → CORS → RateLimit → Handler
```

| 中间件 | 功能 |
|--------|------|
| RequestID | 为每个请求生成唯一ID，添加到响应头 X-Request-ID |
| Logging | 记录访问指标（访问量、用户数），不输出请求日志 |
| CORS | 处理跨域请求，设置 Access-Control 相关响应头 |
| RateLimit | 基于客户端IP的速率限制，防止滥用 |

### 响应头

所有 API 响应都会包含以下头：

| 头名称 | 说明 |
|--------|------|
| X-Request-ID | 请求唯一标识，用于日志追踪 |
| Access-Control-Allow-Origin | 跨域允许来源 |
| Access-Control-Allow-Methods | 支持的HTTP方法 |
| Access-Control-Allow-Headers | 支持的请求头 |

## Token 缓存

系统内置 Token 缓存机制，用于缓存用户登录后的认证 Token，避免频繁登录。

### 缓存策略

- 容量：最多缓存 10000 个 Token
- 过期时间：5 分钟
- 清理：每分钟自动清理过期 Token
- 失效处理：当检测到 Token 失效时（响应 code != "1"），会自动使缓存失效并重新登录

### 缓存键格式

`{providerKey}:{account}`

例如：`hnit_a:2023010101`

### Token 失效重试流程

```
请求课程数据 → 检查响应 code
                      ↓
            code != "1"? → 视为token失效
                      ↓是
      cache.InvalidateToken() 删除缓存
                      ↓
      重新登录获取新token
                      ↓
      替换URL中的token → 重试请求
```

## HTTP 客户端

内置 HTTP 客户端模块 (`internal/httpclient`)，提供：

- 连接池管理：最大 100 个空闲连接，每主机最多 10 个
- 自动重试：默认 3 次重试，针对 5xx 错误和连接问题
- 超时控制：默认 10 秒
- Token 替换：自动从 URL 中提取和替换 Token

## 运行时指标

服务运行时每分钟向控制台输出运行指标：

```
[指标] 访问量: 150 | 用户数: 42 | Token缓存命中率: 85.5%
```

### 指标说明

| 指标 | 说明 |
|------|------|
| 访问量 | 每分钟处理的请求总数（仅包含带 school 参数的请求） |
| 用户数 | 每分钟访问的不同用户（school+account）数量 |
| Token缓存命中率 | Token缓存命中次数 / 总 Token 获取次数 |

所有数据存储在内存中，无磁盘 IO。

## 信号处理

服务支持以下信号：

| 信号 | 行为 |
|------|------|
| SIGINT (Ctrl+C) | 优雅关闭服务器 |
| SIGTERM | 优雅关闭服务器 |

关闭流程：
1. 停止接收新请求
2. 停止 Token 缓存后台清理
3. 停止指标上报
4. 关闭服务器

## 热更新

系统支持配置文件热更新，修改以下文件后会自动重载：

| 文件 | 更新内容 | 重启服务器 |
|------|----------|------------|
| assets/config.json | 服务器端口、超时、速率限制 | 是 |
| assets/school_config.json | 学校学期配置 | 否 |
| assets/404.html | 404 页面内容 | 否 |

## 开发

### 添加新的学校支持

1. 在 `internal/provider/schools/` 下创建新的学校实现文件
2. 实现 `SchoolProvider` 接口：
   ```go
   type SchoolProvider interface {
       GetSchoolId() string
       GetProviderKey() string
       Login(account, password string) (*model.CourseResponse, error)
       GetGrades(account, password, semester string) (*model.CourseResponse, error)
       GetGuidanceTeaching(account, password string) (*model.CourseResponse, error)
   }
   ```
3. 在 `init()` 函数中注册到 Provider Registry
4. 在 `assets/school_config.json` 中添加学校配置

### 运行测试

```bash
go test ./...
```

## 许可证

本项目采用 Apache License, Version 2.0 许可证。详见 LICENSE 文件。

## 更新日志

### v1.0.0 (2026-04-22)
- 初始重构版本
- 支持多学校登录
- 支持配置文件热更新
- 支持 HTTP 连接池
- 支持 Token 缓存自动清理
- 支持请求速率限制
- 支持请求 ID 追踪
- 统一错误处理
