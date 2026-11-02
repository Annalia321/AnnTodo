# AnnTodo

AnnTodo 是一个可自行部署的待办事项（Todo）应用。后端使用 Go（Gin + GORM），前端使用 Vue 3 + Vuetify，前端构建产物通过 `embed` 打包进同一个可执行文件。默认使用 SQLite，一个可执行文件加一个配置文件即可运行。

## 功能

- **列表页**：卡片式分组，条目拖拽排序、跨卡片移动，条目详情面板（备注、截止日期 / 时间、重要 / 紧急、所属卡片），已完成折叠区。
- **日历页**：基于 FullCalendar 的月 / 周 / 日视图，拖拽修改截止时间，按卡片筛选。
- **四象限页**：按重要 / 紧急分为四个象限，可拖拽在象限间移动。
- **设置页**：10 种界面语言、浅色 / 深色 / 跟随系统、主题色、字体、字号、显示密度、时区、日期时间格式、每周第一天；个人资料与头像、修改密码、登录设备管理；数据导入导出与回收站。
- **用户管理**（管理员）：新建用户、重置密码、修改角色、禁用 / 启用、删除；开关新用户注册、修改站点名称。
- **安全**：bcrypt 密码哈希，HttpOnly + SameSite 会话 Cookie，双重提交 Cookie 的 CSRF 防护，登录限流，所有接口校验数据归属。
- **新手引导**：首次登录时用不超过 5 步介绍长按编辑卡片、卡片编辑按钮、四象限筛选等操作。
- **MCP 接入**：在设置页生成 Token 后，AI 客户端可以通过 MCP 读取和修改卡片与待办，详见下文。
- **PWA**：手机浏览器可“添加到主屏幕”，离线时可打开应用外壳。

## 快速开始

需要 Go 1.27+ 和 Node.js 20+（推荐 24）。

```powershell
# 1. 构建前端（产物输出到 web/dist，会被嵌入可执行文件）
cd web
npm ci
npm run build
cd ..

# 2. 构建后端
go build -o anntodo.exe ./cmd/anntodo      # Linux / macOS：-o anntodo

# 3. 运行
.\anntodo.exe
```

浏览器打开 <http://localhost:8080>。首次访问会进入初始化页面，创建的第一个账号就是管理员。

也可以用构建脚本一步完成：

```powershell
.\build.ps1                         # 构建当前平台 -> anntodo.exe
.\build.ps1 -Version 1.0.0 -All     # 交叉编译 Windows / Linux（amd64、arm64）-> dist\
```

```sh
./build.sh                          # 构建当前平台 -> ./anntodo
./build.sh 1.0.0 all                # 交叉编译 -> dist/
```

### 前端开发

```powershell
.\anntodo.exe          # 后端监听 8080
cd web; npm run dev    # Vite 开发服务器 http://localhost:5173，/api 代理到 8080
```

## 命令行

```
anntodo [-c conf/anntodo.ini] [run]                                    启动 Web 服务
anntodo [-c conf/anntodo.ini] service install|uninstall|start|stop|restart|status
anntodo version
```

不指定 `-c` 时，依次查找“可执行文件所在目录/conf/anntodo.ini”和“当前目录/conf/anntodo.ini”。配置文件不存在时使用默认值。

## 配置

配置文件为 [conf/anntodo.ini](conf/anntodo.ini)，各项含义见文件内注释。配置中的相对路径以 `conf` 目录的上一级为基准。

| 配置段 | 主要配置项 |
| --- | --- |
| `[server]` | `host`、`port`、`cert_file` / `key_file`（可选，直接提供 HTTPS）、`trusted_proxies` |
| `[database]` | `type`（`sqlite` / `mysql` / `postgres`）、`path`（SQLite 文件）、`host`、`port`、`name`、`user`、`password`、`ssl_mode` |
| `[storage]` | `data_dir`：数据目录，存放 SQLite 数据库和头像，默认 `data` |
| `[log]` | `level`（`debug` / `info` / `warn` / `error`）、`file`（留空则只输出到控制台） |
| `[session]` | `remember_days`（“记住我”有效天数）、`default_hours`（普通登录的滑动有效期） |

每一项都可以用环境变量覆盖，命名规则为 `ANNTODO_<配置段>_<配置项>`（大写），例如：

```powershell
$env:ANNTODO_SERVER_PORT = "9000"
$env:ANNTODO_DATABASE_TYPE = "postgres"
$env:ANNTODO_DATABASE_PASSWORD = "secret"
```

## 手机访问

服务默认监听 `0.0.0.0:8080`，同一局域网内的手机用浏览器打开 `http://<电脑或服务器的 IP>:8080` 即可。如果打不开，检查以下几点：

- Windows 或 Linux 防火墙（`ufw` / `firewalld`）已放行 8080 端口，云服务器还要在安全组里放行。
- 端口没有被其他程序占用。启动时出现 `bind: Only one usage of each socket address` 说明已有一个 AnnTodo 或其他程序在用这个端口。
- 在公网使用时建议放在反向代理后面并启用 HTTPS，见下文。

## MCP 接入（让 AI 管理待办）

AnnTodo 内置 MCP 服务器（Streamable HTTP），AI 客户端连接后可以调用工具查询和修改当前账号的卡片与待办。

1. 登录后打开 **设置 → MCP 接入**，点击 **生成 Token**。Token 只显示一次，请立即复制保存。
2. 在 AI 客户端（Claude、Cursor、Cherry Studio 等）中添加一个 Streamable HTTP 类型的 MCP 服务器：
   - 地址：`http://<主机>:8080/mcp`（AI 客户端必须能访问到这个地址）
   - 请求头：`Authorization: Bearer sk-...`

设置页可以直接复制完整的 JSON 配置，例如：

```json
{
  "mcpServers": {
    "anntodo": {
      "type": "http",
      "url": "http://192.168.1.10:8080/mcp",
      "headers": { "Authorization": "Bearer sk-..." }
    }
  }
}
```

不同客户端的字段名可能略有差别（例如 `type` 写作 `streamableHttp`），以客户端文档为准。

每个账号同时只有一个 Token，数据库中只保存它的哈希值。重新生成后旧 Token 立即失效；点击 **吊销** 可随时收回访问权限。生成 Token 时会记录浏览器时区，AI 读写的截止日期按这个时区换算。

提供的工具：

| 类别 | 工具 |
| --- | --- |
| 查询 | `get_context`（当前时间、时区、卡片概览）、`list_cards`、`list_items`、`search_items`、`get_item`、`list_trash` |
| 卡片 | `create_card`、`update_card`、`delete_card` |
| 待办 | `create_item`、`update_item`、`complete_item`、`move_item`、`delete_item` |
| 回收站 | `restore_from_trash` |

删除操作会把卡片或待办放入回收站，可以在设置页或通过 `restore_from_trash` 恢复。

## 部署

### Windows 服务

以管理员身份打开终端：

```powershell
.\anntodo.exe service install    # 注册为 Windows 服务（开机自启），会记录当前配置文件的绝对路径
.\anntodo.exe service start
.\anntodo.exe service status
.\anntodo.exe service stop
.\anntodo.exe service uninstall
```

服务以系统账号运行，建议在 `[log] file` 中配置日志文件，方便排查问题。

### Linux（systemd）

```sh
sudo mkdir -p /opt/anntodo && sudo cp -r anntodo conf /opt/anntodo/
sudo useradd --system --home /opt/anntodo --shell /usr/sbin/nologin anntodo
sudo mkdir -p /opt/anntodo/data && sudo chown -R anntodo:anntodo /opt/anntodo
sudo cp docker/anntodo.service /etc/systemd/system/
sudo systemctl daemon-reload
sudo systemctl enable --now anntodo
```

也可以使用 `sudo ./anntodo service install` 自动生成 systemd 服务。

### Docker

```sh
docker build -f docker/Dockerfile -t anntodo --build-arg VERSION=1.0.0 .
docker run -d --name anntodo -p 8080:8080 -v anntodo-data:/app/data --restart unless-stopped anntodo
```

或使用 Compose：

```sh
docker compose -f docker/docker-compose.yml up -d
```

数据目录 `/app/data` 挂载为卷，镜像中的配置可通过 `ANNTODO_*` 环境变量覆盖。国内网络构建时可加 `--build-arg GOPROXY=https://goproxy.cn,direct`。

### 反向代理与 HTTPS

程序本身提供 HTTP 服务，建议通过反向代理对外提供 HTTPS。代理需要传递 `X-Forwarded-For` 和 `X-Forwarded-Proto`，并把代理地址加入 `[server] trusted_proxies`，这样登录限流和设备列表才能拿到真实 IP，Cookie 也会带上 `Secure` 标记。

Nginx 示例：

```nginx
server {
    listen 443 ssl http2;
    server_name todo.example.com;
    ssl_certificate     /etc/ssl/todo.crt;
    ssl_certificate_key /etc/ssl/todo.key;
    client_max_body_size 10m;

    location / {
        proxy_pass http://127.0.0.1:8080;
        proxy_set_header Host $host;
        proxy_set_header X-Forwarded-For $proxy_add_x_forwarded_for;
        proxy_set_header X-Forwarded-Proto $scheme;
    }
}
```

Caddy 示例：

```
todo.example.com {
    reverse_proxy 127.0.0.1:8080
}
```

也可以在配置文件中设置 `cert_file` 和 `key_file`，由程序直接提供 HTTPS。

## 备份与恢复

- **SQLite（默认）**：停止服务后复制整个数据目录（默认 `data/`，包含 `anntodo.db` 和头像）即可。不停服务备份时，要同时复制 `anntodo.db`、`anntodo.db-wal` 和 `anntodo.db-shm`，或使用 `sqlite3 data/anntodo.db ".backup backup.db"`。恢复时把文件放回数据目录后启动服务。
- **MySQL / PostgreSQL**：使用 `mysqldump` 或 `pg_dump` 备份数据库，同时备份数据目录中的头像。
- **个人数据**：每个用户都可以在“设置 → 数据”中导出 JSON，并在其他实例中导入。

## 目录结构

```
cmd/anntodo/    程序入口与命令行
pkg/api/        HTTP 接口与 MCP 服务器
pkg/core/       业务逻辑
pkg/models/     数据模型
pkg/storage/    数据库访问
pkg/settings/   配置读取
pkg/middleware/ 鉴权、CSRF、限流、日志
web/            前端（Vue 3 + Vuetify），web/dist 为构建产物
conf/           默认配置文件
docker/         Dockerfile、docker-compose.yml、systemd 服务示例
```

## 许可证

[MIT](LICENSE)。依赖均为 MIT、BSD、ISC、Apache-2.0 等宽松许可证；MySQL 驱动 `go-sql-driver/mysql` 为 MPL-2.0（文件级弱 copyleft，未修改其源码时可随 MIT 项目分发）。不含 GPL / AGPL 依赖。
