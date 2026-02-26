# 小雨课程表服务端

Copyright 2026 重庆沛雨霖科技有限公司 (PYLinTech)
Contact: PYLinTech@163.com

高并发选课数据获取 API 服务。

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
GET /api/get-support-school
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
GET /api/get-course-data?school=<id>&account=<account>&password=<password>
```

参数：
- `school`: 学校 ID
- `account`: 账号
- `password`: 密码

响应：
```json
{
  "success": true,
  "data": []
}
```

错误响应：
```json
{
  "success": false,
  "msg_zhcn": "不支持的学校: 6",
  "msg_en": "School not supported: 6"
}
```

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
├── assets/             # 静态资源
├── test/               # 测试页面
└── run.sh              # 运行脚本
```
