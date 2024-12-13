# Free Classroom Backend

基于 Go 的空闲教室查询系统后端服务。

## 技术栈

- Go 1.21
- Chi (路由)
- godotenv (环境变量)

## 功能特点

- 提供教室课程表 API
- 自动缓存管理
- 定时刷新数据（每两小时，8:00-21:00）
- CORS 支持
- 环境变量配置

## 开发环境设置

1. 克隆仓库：
```bash
git clone https://github.com/Merryfling/free-classroom-backend.git
cd free-classroom-backend
```

2. 安装依赖：
```bash
go mod tidy
```

3. 创建环境配置文件：
```bash
cp .env.example .env
```

4. 配置环境变量：
   - 编辑 `.env` 文件
   - 设置必要的环境变量（参考 [环境变量](#环境变量) 部分）

5. 启动服务器：
```bash
go run main.go
```

## 环境变量

详细配置请参考 `.env.example` 文件：

- `PORT`: 服务器端口号
- `ALLOWED_ORIGINS`: 允许的前端域名列表
- `UESTC_API_URL`: 教务系统 API 地址
- `CLASSROOMS`: 需要查询的教室列表

## API 接口

### GET /api/schedules

获取所有教室的课程表信息。

响应格式：
```json
{
  "status": 200,
  "message": "success",
  "data": [
    {
      "room": "A101",
      "freeSlots": [1, 2, 3, 4],
      "occupiedSlots": [5, 6, 7, 8]
    }
  ]
}
```

## 部署

### 服务器部署

1. 构建二进制文件：
```bash
go build
```

2. 配置环境变量：
```bash
cp .env.example .env
vim .env  # 编辑环境变量
```

3. 运行服务：
```bash
./free-classroom
```

### 使用 systemd（推荐）

1. 创建服务文件：
```bash
sudo vim /etc/systemd/system/free-classroom.service
```

2. 添加以下内容：
```ini
[Unit]
Description=Free Classroom Backend Service
After=network.target

[Service]
Type=simple
User=your_user
WorkingDirectory=/path/to/free-classroom
ExecStart=/path/to/free-classroom/free-classroom
Restart=always
Environment=PORT=8080
# 添加其他环境变量

[Install]
WantedBy=multi-user.target
```

3. 启动服务：
```bash
sudo systemctl enable free-classroom
sudo systemctl start free-classroom
```

### 使用 Docker（可选）

1. 构建镜像：
```bash
docker build -t free-classroom-backend .
```

2. 运行容器：
```bash
docker run -d \
  -p 8080:8080 \
  --env-file .env \
  --name free-classroom \
  free-classroom-backend
```

## 项目结构

```
backend/
├── main.go          # 主程序入口
├── go.mod           # Go 模块定义
└── .env.example     # 环境变量示例
```

## 贡献

欢迎提交 Pull Request 和 Issue。

## 许可证

MIT