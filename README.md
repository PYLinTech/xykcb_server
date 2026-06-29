# xykcb_server

小雨课程表服务端的腾讯云 EdgeOne Makers Go Cloud Functions 版本，采用官方推荐的 **Handler 模式**，用于向客户端提供学校列表、功能入口、课程、成绩和培养方案等接口。

## 功能

- 查询支持学校
- 查询学校功能入口
- 获取课程 TSV 数据
- 获取成绩
- 获取培养方案
- 南华大学验证码模板识别支持
- `hnit_b` 内网场景固定返回不支持码

## 项目结构

```text
xykcb_server/
├── edgeone.json
├── README.md
├── LICENSE
├── NOTICE
├── GitRules.md
└── cloud-functions/
    ├── go.mod
    ├── [[default]].go
    └── ocr_templates.gz
```

## EdgeOne Go 约定

- `cloud-functions/[[default]].go` 使用 EdgeOne 多级动态路由写法，承接根路径下的旧版 API，例如 `/get-support-school`、`/get-course-data`。
- Go 文件使用 `package handler`，并导出 `Handler(w http.ResponseWriter, r *http.Request)`，签名符合 `http.HandlerFunc`。
- `go.mod` 放在 `cloud-functions/` 目录内。
- `edgeone.json` 使用 `cloudFunctions.go.maxDuration` 设置 Go 函数最大运行时长。
- 南华大学验证码模板文件 `cloud-functions/ocr_templates.gz` 通过 `cloudFunctions.nodejs.includeFiles` 和 `cloudFunctions.go.includeFiles` 保留到函数构建产物中，代码运行时会从函数工作目录、可执行文件目录和常见部署目录查找，避免 `go:embed` 在 EdgeOne 单文件编译流程中找不到资源。

> 当前 EdgeOne 文档把 `includeFiles` 放在 `cloudFunctions.nodejs` 下，但实际构建器会据此复制函数运行期需要读取的文件。本项目保留该字段，因为部署验证已通过，并且这是保留外部 OCR 模板文件的最小侵入方式。
> 同时保留 `cloudFunctions.go.includeFiles`，用于兼容 Go 构建器支持该字段的部署环境。

## API 协议

所有业务接口均使用 `GET` 请求。默认返回 `application/json; charset=utf-8`，未定义路径返回 404 HTML。

| 路径 | 说明 |
|------|------|
| `/get-support-school` | 获取支持学校列表 |
| `/get-support-function?school=<school>` | 获取指定学校支持的功能 |
| `/get-course-data?school=<school>&account=<account>&password=<password>&semester=<semester>` | 获取课程 TSV 数据 |
| `/get-course-grades?school=<school>&account=<account>&password=<password>&semester=<semester>` | 获取成绩 |
| `/get-guidance-teaching?school=<school>&account=<account>&password=<password>` | 获取培养方案 |

账号密码参数兼容两种写法：

| 新参数 | 兼容旧参数 | 说明 |
|--------|------------|------|
| `account` | `student_ID` | 学号/账号 |
| `password` | `student_password` | 密码 |

## 支持学校

| 学校代码 | 学校 | 说明 |
|----------|------|------|
| `hnit_a` | 湖南工学院 | 外网教务入口 |
| `hnit_b` | 湖南工学院 | 内网教务入口，服务器不支持，固定返回 `002` |
| `hynu` | 衡阳师范学院 | 外网教务入口 |
| `usc` | 南华大学 | 外网教务入口，包含验证码识别模板 |

## 通用响应格式

成功：

```json
{
  "success": true,
  "data": {}
}
```

失败：

```json
{
  "success": false,
  "desc_key": "001"
}
```

不同接口的 `data` 结构会根据客户端协议返回。课程接口通常返回课程 TSV 文本或经过封装后的课程数据；成绩和培养方案接口按学校教务系统解析结果返回。

## 错误码

| desc_key | 说明 |
|----------|------|
| `001` | 请求参数或请求方法错误 |
| `002` | 学校或功能不支持 |
| `003` | 账号或密码错误 |
| `004` | 教务系统异常、验证码失败、网络错误或服务端处理失败 |

说明：`hnit_b` 属于服务器无法访问内网教务的固定场景，课程、成绩和培养方案接口均返回 `002`。


## EdgeOne 配置

`edgeone.json`：

```json
{
  "name": "xykcb-server",
  "cloudFunctions": {
    "nodejs": {
      "includeFiles": [
        "cloud-functions/ocr_templates.gz"
      ]
    },
    "go": {
      "maxDuration": 120,
      "includeFiles": [
        "cloud-functions/ocr_templates.gz"
      ]
    }
  }
}
```

## 部署

推荐使用 EdgeOne Makers 项目直接构建部署。

本地调试：

```bash
edgeone makers dev
```

部署流程：

1. 将项目上传或推送到 EdgeOne Makers 项目。
2. 保持 `cloud-functions/` 位于项目根目录。
3. 确认 `edgeone.json` 位于项目根目录。
4. 触发 EdgeOne Makers 构建部署。

## 本地检查

`[[default]].go` 是 EdgeOne 的动态路由文件名，原生 `go test ./...` 会认为文件名非法；如需做纯 Go 语法检查，可临时复制为普通文件名再执行：

```bash
cd cloud-functions
cp '[[default]].go' main.go
go test ./...
rm main.go
```

## 数据与安全

- 本项目仅作为客户端和学校教务系统之间的协议适配服务。
- 项目不会主动持久化账号、密码、课表、成绩或培养方案数据。
- 账号密码仅用于本次请求中的教务系统登录流程。
- 建议生产环境使用 HTTPS 域名访问。
- 不建议在日志、截图或公开 Issue 中暴露账号、密码、Cookie、验证码、Token 等敏感信息。

## License

本项目基于 Apache License 2.0 发布。完整协议见 `LICENSE`。

## NOTICE

项目版权、运行平台和声明信息见 `NOTICE`。

## 第三方平台与商标声明

Tencent EdgeOne、EdgeOne Pages、EdgeOne Makers 等名称、商标和服务归其各自权利人所有。本项目仅适配其 Go Cloud Functions 运行环境，不代表与平台方存在官方合作、背书或从属关系。

各学校名称、教务系统名称及相关标识归其各自权利人所有。本项目仅用于课程表客户端的个人数据查询适配，不代表与相关学校存在官方合作、背书或从属关系。

## 免责声明

本项目仅用于学习、研究和个人课程表数据同步。使用者应遵守学校教务系统、网络服务和相关法律法规要求。因账号使用、网络请求、数据同步、接口变更或部署配置产生的后果，由使用者自行承担。
