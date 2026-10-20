# chat-app — Go 聊天应用（学习项目）

一个**功能完整的 WebSocket 聊天应用**，从零实现了用户认证、群聊/私聊、实时消息推送、在线状态、站内通知、文件上传等完整闭环。**本项目定位为 Go 语言学习项目**，代码刻意保持分层清晰、注释完整，适合初学者通读。

## 功能特性

- 🔐 邮箱验证码注册 / 登录 / 忘记密码重置（验证码存 Redis，5 次错误自动作废）
- 💬 群聊 + 私聊，文本 / 图片 / 文件消息
- ⚡ WebSocket 实时推送（心跳保活、断线重连、join/leave 房间订阅）
- 🟢 在线状态（Redis 实时状态 + 上线/离线广播）
- 🔔 站内通知（离线消息上线推送、通知已读）
- 👤 用户资料（昵称 / 头像，头像上传）
- 📁 文件上传（扩展名白名单、按日期分目录、10MB 上限）
- 🌐 原生 ES Modules 前端（零依赖，同源部署免跨域）

## 技术栈

| 分类 | 选型 | 作用 |
|---|---|---|
| Web 框架 | [Gin](https://github.com/gin-gonic/gin) | HTTP 路由 / 中间件 / 参数绑定 |
| ORM | [GORM](https://gorm.io) + MySQL | 数据模型与 CRUD |
| 缓存 | [go-redis/v8](https://github.com/go-redis/redis) | 验证码、在线状态 |
| 认证 | [golang-jwt/v5](https://github.com/golang-jwt/jwt) | 登录态签发与校验 |
| 实时通信 | [gorilla/websocket](https://github.com/gorilla/websocket) | WebSocket 协议 |
| 配置 | [viper](https://github.com/spf13/viper) | YAML 配置加载 |
| 密码加密 | [golang.org/x/crypto/bcrypt](https://pkg.go.dev/golang.org/x/crypto/bcrypt) | 密码哈希存储 |

## 快速开始

### 前置要求

- Go 1.26+
- MySQL 5.7+（需支持 utf8mb4）
- Redis 6+

### 步骤

```bash
# 1. 克隆 / 下载项目后进入目录
cd chat-app

# 2. 复制配置并填写（数据库、Redis、SMTP、JWT 密钥）
#    所有配置都在 config/config.yaml，每项都有注释说明
vim config/config.yaml

# 3. 建库（数据库名与 config.yaml 的 dbname 一致，首次启动会自动建表）
mysql -u root -p -e "CREATE DATABASE chat_db CHARACTER SET utf8mb4 COLLATE utf8mb4_unicode_ci;"

# 4. 启动
go run ./cmd/server
```

启动成功输出 `聊天服务器启动在 ws://localhost:8080`，浏览器访问 <http://localhost:8080> 即可使用（前端已由后端静态托管，无需单独启动前端）。

> 注册流程需要邮箱验证码——请在 `config.yaml` 的 `email` 段填入你的 SMTP 邮箱（QQ/163 邮箱需使用**授权码**，不是登录密码）。

## 目录结构（重点：分层架构）

```
chat-app/
├── cmd/server/main.go          # 入口：装配依赖、注册路由、启动服务
├── config/config.yaml          # 全部配置（数据库/Redis/JWT/邮件/上传）
├── internal/                   # 业务代码（对外不可导入，Go 约定）
│   ├── model/                  # 数据模型（User/Room/Message/...，即数据库表）
│   ├── repository/             # 数据访问层：只做 SQL/Redis 读写，不写业务逻辑
│   ├── service/                # 业务逻辑层：校验、组合、规则（如验证码校验、成员权限）
│   ├── handler/                # HTTP 层：参数解析、调用 service、返回 JSON
├── pkg/                        # 可复用基础能力（对外可导入）
│   ├── auth/                   # JWT 签发与认证中间件
│   ├── websocket/              # WS 连接管理（Hub/Client）、消息协议
│   ├── email/                  # SMTP 邮件发送
│   ├── config/                 # 配置加载（viper）
│   └── upload/                 # 文件上传服务抽象（本地实现）
├── web/                        # 前端工程（原生 ES Modules，零依赖）
└── test_api.sh                 # 接口测试脚本（仅需 curl）
```

### 为什么这样分层？（学习重点）

依赖方向**单向流动**：`handler → service → repository → model`

```
HTTP 请求 → handler（参数解析）
                ↓
           service（业务规则：验证码对不对？是不是房间成员？）
                ↓
           repository（SQL/Redis 读写）
                ↓
           model（数据表结构）
```

- **handler** 只做"翻译"：把 HTTP 参数变成 Go 结构体，把业务错误变成 HTTP 状态码
- **service** 只做"决策"：业务规则全部在这层（比如"群聊消息必须校验发送者是房间成员"）
- **repository** 只做"存取"：一行业务逻辑都不写，可替换数据库实现
- **model** 只是"结构"：对应数据库表，字段带 json 标签

这样改任何一层都不影响其他层——比如把 MySQL 换成 PostgreSQL，只动 repository 和 config。

## 接口文档

### 统一响应格式

```json
{ "success": true, "message": "操作成功", "data": { } }
```

| HTTP 状态码 | 含义 |
|---|---|
| 200 | 成功 |
| 400 | 参数错误 / 验证码错误 / 文件超限 |
| 401 | 未登录或密码错误（token 失效） |
| 403 | 不是房间成员，无权操作 |
| 404 | 用户 / 房间 / 消息不存在 |
| 409 | 邮箱已注册 / 已是房间成员 |

分页接口的 `data` 统一为：`{ "items": [...], "total": 100, "page": 1, "page_size": 20 }`

### 认证模块（无需登录）

| 方法 | 路径 | 参数 | 说明 |
|---|---|---|---|
| GET | `/health` | - | 健康检查 |
| POST | `/api/auth/email-code` | `email` | 发送注册验证码（存 Redis，10 分钟有效） |
| POST | `/api/auth/register` | `username, email, password, code`（`nickname` 可选） | 邮箱验证码注册 |
| POST | `/api/auth/login` | `email, password` | 登录，返回 `{ token, user }` |
| POST | `/api/auth/password/reset-code` | `email` | 发送密码重置验证码 |
| POST | `/api/auth/password/reset` | `email, code, new_password` | 重置密码 |

### 用户模块（需 JWT）

| 方法 | 路径 | 参数 | 说明 |
|---|---|---|---|
| GET | `/api/user/profile` | - | 获取当前用户资料 |
| PUT | `/api/user/profile` | `nickname` / `avatar`（至少一个） | 更新昵称 / 头像 |

### 房间模块（需 JWT）

| 方法 | 路径 | 参数 | 说明 |
|---|---|---|---|
| POST | `/api/rooms` | `name`（`description/type/avatar` 可选） | 创建房间（创建者自动成为 owner） |
| GET | `/api/rooms` | - | 我的房间列表（含成员数） |
| GET | `/api/rooms/:id` | - | 房间详情 |
| POST | `/api/rooms/:id/join` | - | 加入房间 |
| POST | `/api/rooms/:id/leave` | - | 退出房间（owner 不可退） |
| GET | `/api/rooms/:id/members` | - | 成员列表（含用户信息与角色） |

### 消息模块（需 JWT）

| 方法 | 路径 | 参数 | 说明 |
|---|---|---|---|
| POST | `/api/messages` | `receiver_id` 或 `room_id`（二选一）、`content`、`type: text/image/file` | 发送消息（群聊校验成员身份） |
| GET | `/api/messages` | `target_id` 或 `room_id`、`page`、`page_size` | 查询消息（私聊 / 群聊历史） |
| GET | `/api/messages/unread-count` | - | 未读消息数 |
| POST | `/api/messages/:id/read` | - | 标记已读（仅接收者本人） |

### 在线状态 / 通知 / 上传（需 JWT）

| 方法 | 路径 | 参数 | 说明 |
|---|---|---|---|
| GET | `/api/users/online` | - | 在线用户 ID 列表（Redis 实时） |
| GET | `/api/users/:id/presence` | - | 指定用户在线状态 |
| GET | `/api/notifications` | `page`、`page_size` | 我的通知列表 |
| POST | `/api/notifications/:id/read` | - | 标记通知已读 |
| POST | `/api/upload` | `file`（multipart） | 上传文件，返回 `{ url, name, size }` |

### 认证方式（JWT）

登录后拿到 `token`，后续请求带在请求头：

```
Authorization: Bearer <token>
```

WebSocket 连接无法自定义请求头（浏览器限制），改为 query 传参：

```
ws://localhost:8080/ws?token=<token>
```

### 接口调用示例（curl）

```bash
# 登录
curl -X POST http://localhost:8080/api/auth/login \
  -H "Content-Type: application/json" \
  -d '{"email":"you@example.com","password":"123456"}'

# 带 token 获取资料
curl http://localhost:8080/api/user/profile \
  -H "Authorization: Bearer <token>"

# 创建房间
curl -X POST http://localhost:8080/api/rooms \
  -H "Authorization: Bearer <token>" \
  -H "Content-Type: application/json" \
  -d '{"name":"Go 学习群"}'

# 发群聊消息
curl -X POST http://localhost:8080/api/messages \
  -H "Authorization: Bearer <token>" \
  -H "Content-Type: application/json" \
  -d '{"room_id":1,"content":"大家好","type":"text"}'
```

## WebSocket 协议

连接：`GET /ws?token=<JWT>`

**客户端 → 服务端**：

```json
{ "type": "ping" }                          // 心跳（建议每 25s 一次）
{ "type": "join", "room_id": 1 }            // 订阅群聊房间
{ "type": "leave", "room_id": 1 }           // 取消订阅
```

**服务端 → 客户端**：

```json
{ "type": "message", "msg_type": "text", "id": 1, "sender_id": 2, "room_id": 1, "content": "hi", "created_at": "..." }
{ "type": "presence", "user_id": 2, "status": "online" }
{ "type": "notification", "title": "...", "content": "...", "is_read": false }
{ "type": "pong" }
```

## 关键设计（学习点）

1. **密码安全**：bcrypt 哈希存储（`golang.org/x/crypto/bcrypt`），数据库不存明文
2. **验证码**：存 Redis 而非数据库——短时效数据天然过期（10 分钟 TTL），附错误次数限制（5 次作废防爆破）。键设计：`email:code:register:<email>`、`email:attempts:register:<email>`
3. **JWT 认证**：签发后服务端无状态，Claims 带 `{UserID, Username, Role}`，有效期 24h（config 可配）
4. **在线状态**：Redis `user:status:<id>` + TTL 5 分钟心跳续期，离线自动清除；多端登录最后一个连接断开才置离线
5. **消息实时推送**：发送 → 落库 → WebSocket 广播；REST 历史与 WS 实时帧字段对齐（`msg_type`）
6. **群聊权限**：发消息、查历史前校验"是房间成员"（service 层 `roomRepo.IsMember`）
7. **上传安全**：扩展名白名单 + 重命名（时间戳+随机数）+ 按日期分目录，杜绝路径注入
8. **前端零依赖**：原生 ES Modules + hash 路由 + 订阅发布式状态管理，同源部署免跨域

## 测试

```bash
# 接口测试脚本（零依赖，仅需 curl；验证码可交互输入或用 REG_CODE_* 环境变量预设）
./test_api.sh
```

前端接口文档（Apifox 导入用）：`chat-app.openapi.json`（OpenAPI 3.0，20 个接口 + 19 个数据模型）

## 部署

- **Docker**：`docker compose up -d --build`（多阶段构建，已配国内镜像源加速依赖下载）
- **Nginx 反代**：参考 `deploy/nginx.conf`（WebSocket 需 `Upgrade/Connection` 升级头）
- 详细说明见 `web/README.md`

## 给新人的阅读建议

1. 先读 `cmd/server/main.go`——看一个 Go Web 服务怎么"组装"起来的
2. 再读 `internal/model/` 和 `internal/repository/`——理解数据从哪来
3. 然后读 `internal/service/` 和 `internal/handler/`——理解业务规则在哪
4. 接着读 `pkg/auth/` 和 `pkg/websocket/`——理解认证与实时通信
5. 最后对照本文档的接口表，用 curl 或 Apifox 实际调一遍，走通注册 → 登录 → 建群 → 发消息 → 收实时消息的完整流程

---


