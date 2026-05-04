# 小雨课程表服务端

小雨课程表服务端是一个 Go HTTP API 服务，用于获取学校课程、成绩和培养方案数据。当前主要实现 `hnit_a`，其他学校保留兼容存根。

## 版权信息

Copyright 2026 重庆沛雨霖科技有限公司 (PYLinTech)

Contact: PYLinTech@163.com

## 功能

| 功能 | 说明 |
|------|------|
| 课程数据 | 获取 HNIT 移动端课表数据，输出 TSV 格式课程表 |
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

### 响应格式

除 `/get-course-data` 外，其余接口统一返回 JSON：

```json
{
  "success": false,
  "desc_key": "001"
}
```

`/get-course-data` 成功时 `data` 为 TSV 字符串：

```json
{
  "success": true,
  "data": "@terms\nschool_id\tterm_id\t..."
}
```

### 错误码

| desc_key | HTTP 状态码 | 说明 |
|----------|-------------|------|
| 001 | 400 | 请求参数错误 |
| 002 | 404 | 不支持的学校 |
| 003 | 401 | 账户或密码错误 |
| 004 | 500 | 服务器内部错误 |
| 005 | 429 | 频率超限 |

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

返回 TSV 格式课程表，包含两个表段：`@terms` 学期与节次配置、`@courses` 课程时段明细。

TSV 通用规则：
- 空白单元格表示继承上一行同列还原后的值
- `\N` 表示真正的空值
- `c_hash` 由所有字段的还原后完整值计算 MurmurHash3 32-bit → Base36 左补 0 至 8 位
- 任一课程时段字段不同则必须单独一行

#### @terms — 学期与节次配置

| 字段 | 说明 |
|------|------|
| school_id | 学校 ID |
| term_id | 学期 ID |
| total_weeks | 总周数 |
| start_date | 第 1 周周一日期，ISO 格式 |
| period_group | 大节编号 |
| section_no | 小节编号 |
| section_start_time | 小节开始时间 |
| section_end_time | 小节结束时间 |

#### @courses — 课程时段明细

| 字段 | 说明 |
|------|------|
| c_hash | 课程时段哈希 ID |
| term_id | 学期 ID，关联 @terms.term_id |
| raw_id | 原始课程 ID |
| course_name | 课程名 |
| location | 地点 |
| teacher | 教师 |
| weeks | 上课周次，逗号分隔数字 |
| weekday | 星期，1-7，1 为周一 |
| sections | 节次，逗号分隔数字 |

响应示例：

```
@terms
school_id	term_id	total_weeks	start_date	period_group	section_no	section_start_time	section_end_time
hnit_a	2024-2025-2	20	2025-02-17	1	1	08:30	09:15
					2	09:20	10:05
				2	3	10:25	11:10
					4	11:15	12:00
	2025-2026-1		2025-09-01	1	1	08:30	09:15
					2	09:20	10:05

@courses
c_hash	term_id	raw_id	course_name	location	teacher	weeks	weekday	sections
01e6y2go	2024-2025-2	F0233264	计算机组成原理	1501	T001	2,3,4,5,7,8,9,10,11	2	7,8
00r2k7sc		B0101114	思想政治理论课（一）	2604	T002	2,3,4,5,7,8,9,10,11,12	\N	3,4
01abcdef						3	1,2
00ghijkl	2025-2026-1	F0170514	软件工程	2611	T010	2,3,4,5,6,7,8,9,10,11,12,13	4	1,2
```

`c_hash` 计算规则：对 `term_id`、`raw_id`、`course_name`、`location`、`teacher`、`weeks`、`weekday`、`sections` 的还原后完整值用 `\t` 拼接，计算 MurmurHash3 32-bit（seed=0），结果转 Base36 字符串左补 0 至 8 位。`\N` 参与计算前还原为空字符串。

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
    "semesterConfigTTL": 86400,
    "semesterConfigFrom": [
      {
        "from": "2024-2025-2",
        "totalWeeks": 20,
        "mergeableSections": ["1-2", "3-4", "5-6", "7-8", "9-10"],
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
| mergeableSections | 可合并节次，用于派生 TSV `period_group` |
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
