# xykcb-server

小雨课程表服务端，部署于 Tencent EdgeOne Pages Edge Functions。

## 功能

- 查询支持学校
- 查询学校功能入口
- 获取课程 TSV 数据
- 获取成绩
- 获取培养方案

## API

所有接口使用 `GET`。

| 路径 | 说明 |
|------|------|
| `/get-support-school` | 支持学校 |
| `/get-support-function?school=hnit_a` | 学校功能 |
| `/get-course-data?school=hnit_a&account=<account>&password=<password>` | 课程数据 |
| `/get-course-grades?school=hnit_a&account=<account>&password=<password>&semester=<semester>` | 成绩 |
| `/get-guidance-teaching?school=hnit_a&account=<account>&password=<password>` | 培养方案 |

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

错误码：

| desc_key | 说明 |
|----------|------|
| `001` | 请求参数或方法错误 |
| `002` | 学校不支持 |
| `003` | 账号或密码错误 |
| `004` | 教务系统或服务端处理失败 |

未定义路径返回 404 HTML。

## 结构

```text
edge-functions/
├── index.js
├── [[default]].js
└── _shared/
    ├── hnit-a.js
    ├── http.js
    ├── not-found.js
    ├── schools.js
    └── tsv.js
```

## 配置

`edgeone.json`：

```json
{
  "name": "xykcb-server",
  "buildCommand": "npm run build",
  "installCommand": "npm install",
  "nodeVersion": "22.11.0"
}
```

## 部署

```bash
npm install
npm run build
edgeone pages deploy
```

## License

Apache License 2.0. See `LICENSE` and `NOTICE`.
