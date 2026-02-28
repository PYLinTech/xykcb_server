# 小雨课程表服务端

Copyright 2026 重庆沛雨霖科技有限公司 (PYLinTech)
Contact: PYLinTech@163.com

高并发选课数据获取 API 服务。

## 功能特性

- 支持多学校登录
- 配置文件热更新
- 服务器优雅关闭

## 支持学校

| ID | 中文名称 | 英文名称 |
|----|----------|----------|
| 1 | 湖南工学院（移动端） | Hunan Institute Of Technology (Mobile) |
| 2 | 湖南工学院（PC端） | Hunan Institute Of Technology (PC) |
| 3 | 衡阳师范学院 | Hengyang Normal University |
| 4 | 南华大学 | University of South China |

## API 接口

### 获取支持学校列表

```
GET /get-support-school
```

响应：
```json
{
  "success": true,
  "data": [
    {"id": "1", "name_zhcn": "湖南工学院（移动端）", "name_en": "Hunan Institute Of Technology (Mobile)"},
    {"id": "2", "name_zhcn": "湖南工学院（PC端）", "name_en": "Hunan Institute Of Technology (PC)"},
    {"id": "3", "name_zhcn": "衡阳师范学院", "name_en": "Hengyang Normal University"},
    {"id": "4", "name_zhcn": "南华大学", "name_en": "University of South China"}
  ]
}
```

### 获取课程数据

```
GET /get-course-data?school=<id>&account=<account>&password=<password>
```

参数：
- `school`: 学校 ID
- `account`: 账号
- `password`: 密码

响应（湖南工学院移动端）：
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
      "courses": [
        {
          "id": "A001",
          "name": "高等数学",
          "location": "教学楼A101",
          "teacher": "张三",
          "weeks": [1,2,3,4,5,6,7,8,9,10,11,12,13,14,15,16,17,18],
          "schedule": {"1": [1,2], "3": [3,4]}
        }
      ]
    }
  }
}
```

错误响应：
```json
{
  "success": false,
  "msg_zhcn": "账号或密码错误",
  "msg_en": "Invalid account or password"
}
```

## 配置文件

### assets/config.json

```json
{
  "server": {
    "port": "8080",
    "httpReadTimeout": 30,
    "httpWriteTimeout": 30
  },
  "cors": {
    "allowAll": true,
    "allowedHosts": []
  }
}
```

| 字段 | 类型 | 说明 |
|------|------|------|
| server.port | string | 服务器端口，默认 8080 |
| server.httpReadTimeout | int | HTTP 读取超时（秒），默认 30 |
| server.httpWriteTimeout | int | HTTP 写入超时（秒），默认 30 |
| cors.allowAll | bool | 是否允许所有域名跨域，true 为允许，false 为只允许 specifiedHosts |
| cors.allowedHosts | array | 允许跨域的域名列表，当 allowAll 为 false 时生效，格式：`["https://api.pylin.cn", "https://xykcb.pylin.cn"]` |

### assets/school_config.json

学校配置 JSON 文件，可配置各学校的学期信息、时间段等。

## 快速开始

```bash
./run.sh
```

服务器将在 http://localhost:8080 启动。

## 项目结构

```
.
├── cmd/server/          # 入口程序
├── internal/
│   ├── config/          # 配置
│   ├── handler/         # HTTP 处理器
│   ├── model/           # 数据模型
│   └── provider/        # 学校提供者
│       └── schools/     # 各学校实现
├── assets/             # 静态资源（配置、404页面）
└── run.sh              # 运行脚本
```
