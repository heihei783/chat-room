# chat-app Web 前端

原生 ES Modules 实现（**零依赖、无需 npm install**），浏览器直接运行。工程按企业级分层组织，依赖单向流动：

```
web/
├── index.html            # 入口页面（#app 视图挂载点 + #overlay 浮层挂载点）
├── assets/               # 静态资源（logo 等）
├── styles/               # 样式分层：reset（重置）/ variables（设计令牌）/ global（通用）/ auth / chat
└── js/
    ├── config/           # 环境配置（API/WS 地址，可被 window.__APP_ENV__ 覆盖）
    ├── utils/            # 通用工具：storage / format / validate / dom（无业务依赖）
    ├── api/              # 接口层：http 封装（token 注入、统一错误）+ 8 个领域模块
    ├── services/         # 业务服务层：组合 api 与 store，页面唯一依赖
    ├── store/            # 全局状态：订阅-发布，跨组件共享数据唯一来源
    ├── router/           # hash 路由 + JWT 登录守卫
    ├── websocket/        # WS 封装：心跳、断线重连、消息分发
    ├── components/       # 8 个可复用组件（Toast/Modal/Avatar/房间/消息/输入/成员/通知）
    ├── views/            # 页面：登录 / 注册 / 忘记密码 / 聊天主页面
    └── main.js           # 入口：路由启动 + 登录态恢复
```

## 分层依赖规则

```
views → services → api → utils/config
     ↘ components ↗ store（订阅）
websocket → services / store（分发实时消息）
```

- 页面不直接调 api，只依赖 services
- 组件不直接调 api，只通过回调与 store 交互（MessageInput 上传除外，走 uploadService）
- 全局状态只经 store 读写，杜绝组件间私有传参

## 运行方式

**方式一（推荐，同源免跨域）**：后端 `main.go` 已注册 `r.Static("/", "./web")`，启动后端后直接访问：

```
http://localhost:8080
```

**方式二（纯静态预览）**：在 web 目录起静态服务，然后浏览器打开（注意此时跨域调用 API 会被拦截，仅适合页面预览）：

```
cd web
python -m http.server 8000
# 访问 http://localhost:8000
```

## 功能覆盖（对接后端全部接口）

- 认证：邮箱验证码注册 / 登录 / 忘记密码重置（60s 验证码倒计时）
- 房间：创建 / 按 ID 加入 / 退出 / 成员列表 / 邀请（复制房间 ID）
- 消息：私聊与群聊、文本与图片消息（Enter 发送）、历史分页加载、已读回执、未读角标
- 实时：WebSocket 心跳保活、断线指数退避重连、join/leave 房间订阅、实时消息/在线状态/通知推送
- 在线状态：在线用户列表、成员在线标识，点击成员/在线用户可发起私聊
- 通知：站内通知中心（未读角标、全部已读）
- 用户：资料卡、退出登录

## 接口与状态说明

- 统一响应 `{ success, message, data }`；http.js 已封装 token 注入与错误提示，业务失败抛 `Error(message)`
- WebSocket 地址默认 `ws://localhost:8080/ws?token=<JWT>`（浏览器原生 WS 无法自定义请求头）
- 如后端部署在其它地址，在 index.html 内联 `window.__APP_ENV__ = { API_BASE, WS_BASE }` 覆盖
