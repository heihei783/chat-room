package main

import (
	"chat-app/internal/handler"
	"chat-app/internal/repository"
	"chat-app/internal/service"
	"chat-app/pkg/auth"
	"chat-app/pkg/config"
	"chat-app/pkg/email"
	"chat-app/pkg/websocket"
	"fmt"
	"log"
	"net/http"
	"strings"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/go-redis/redis/v8"
	"gorm.io/gorm"
)

func main() {
	// 加载配置
	cfg, err := config.LoadConfig("./config")
	if err != nil {
		log.Fatal("加载配置失败:", err)
	}

	// 解析JWT令牌有效期，配置无效时默认24小时
	tokenTTL, err := time.ParseDuration(cfg.JWT.Expire)
	if err != nil || tokenTTL <= 0 {
		tokenTTL = 24 * time.Hour
	}

	// 初始化JWT（密钥来自配置）
	auth.InitJWT(cfg.JWT.Secret)

	// 初始化数据库
	dsn := repository.BuildDSN(cfg.Database)
	db, err := repository.InitDB(dsn)
	if err != nil {
		log.Fatal("初始化数据库失败:", err)
	}

	// 初始化Redis客户端（用于在线状态缓存，验证码等场景也可复用）
	redisClient := redis.NewClient(&redis.Options{
		Addr:     fmt.Sprintf("%s:%s", cfg.Redis.Host, cfg.Redis.Port),
		Password: cfg.Redis.Password,
		DB:       cfg.Redis.DB,
	})

	// 配置WebSocket（握手超时、心跳周期、消息大小上限均取自config.yaml）
	websocket.Configure(
		cfg.WebSocket.ReadBufferSize,
		cfg.WebSocket.WriteBufferSize,
		cfg.WebSocket.HandshakeTimeout,
		cfg.WebSocket.PingPeriod,
		cfg.WebSocket.PongWait,
		cfg.WebSocket.MaxMessageSize,
		nil,
	)

	// 创建WebSocket Hub并初始化（连接管理与在线状态判断共享同一实例）
	hub := websocket.NewHub()
	handler.InitHub(hub)

	// 设置 Gin 模式
	gin.SetMode(cfg.Server.Mode)

	// 创建 Gin 引擎
	r := gin.Default()

	// 按业务模块注册路由
	registerCommonRoutes(r)
	registerAuthRoutes(r, db, redisClient, cfg, tokenTTL)
	registerRoomRoutes(r, db)
	registerMessageRoutes(r, db, hub)
	registerPresenceRoutes(r, db, hub, redisClient)
	registerNotificationRoutes(r, db, hub)
	registerUploadRoutes(r, cfg)

	// 静态托管前端：所有未匹配 API/WebSocket 的请求回退到 web 目录
	registerWebRoutes(r)

	// 启动服务器
	addr := fmt.Sprintf("%s:%s", cfg.Server.Host, cfg.Server.Port)
	fmt.Printf("聊天服务器启动在 ws://%s\n", addr)
	if err := r.Run(addr); err != nil {
		log.Fatal("启动服务器失败:", err)
	}
}

// registerCommonRoutes 注册全局公共路由：健康检查与WebSocket端点
func registerCommonRoutes(r *gin.Engine) {
	// 健康检查
	r.GET("/health", func(c *gin.Context) {
		c.JSON(200, gin.H{
			"status":  "ok",
			"message": "服务运行正常",
		})
	})

	// WebSocket端点（需JWT认证：浏览器客户端以 /ws?token=<jwt> 方式连接）
	r.GET("/ws", auth.AuthMiddleware(), handler.HandleWebSocket)
}

// registerAuthRoutes 注册认证与用户模块路由（无需登录：注册/登录/发验证码；需JWT：资料查询与修改）
// 用户注册依赖邮件服务做验证码校验，两个模块的依赖在此一并组装
func registerAuthRoutes(r *gin.Engine, db *gorm.DB, redisClient *redis.Client, cfg *config.Config, tokenTTL time.Duration) {
	// 邮件模块：验证码仓储（Redis，短时效数据TTL自动过期）→ 邮件服务（SMTP发送器）→ 邮件处理器
	emailCodeRepo := repository.NewEmailCodeRepository(redisClient)
	emailSender := email.NewSender(email.Config{
		Host:     cfg.Email.Host,
		Port:     cfg.Email.Port,
		Username: cfg.Email.Username,
		Password: cfg.Email.Password,
		From:     cfg.Email.From,
		SSL:      cfg.Email.SSL,
	})
	emailService := service.NewEmailService(emailCodeRepo, emailSender)
	emailHandler := handler.NewEmailHandler(emailService)

	// 用户模块：用户仓储 → 用户服务（依赖邮件服务做验证码校验）→ 用户处理器
	userRepo := repository.NewUserRepository(db)
	userService := service.NewUserService(userRepo, emailService, tokenTTL)
	userHandler := handler.NewUserHandler(userService)

	// 定时清理过期会话（每小时一次），防止user_sessions表无限增长
	go func() {
		ticker := time.NewTicker(time.Hour)
		defer ticker.Stop()
		for range ticker.C {
			_ = userRepo.DeleteExpiredSessions()
		}
	}()

	// 认证路由（无需登录即可访问）
	authGroup := r.Group("/api/auth")
	{
		// 发送邮箱验证码
		authGroup.POST("/email-code", emailHandler.SendEmailCode)
		// 注册（依据邮箱+验证码）
		authGroup.POST("/register", userHandler.Register)
		// 登录（依据邮箱+密码）
		authGroup.POST("/login", userHandler.Login)
		// 发送密码重置验证码
		authGroup.POST("/password/reset-code", emailHandler.SendResetCode)
		// 重置密码（忘记密码场景，验证码+新密码）
		authGroup.POST("/password/reset", userHandler.ResetPassword)
	}

	// 用户路由（需要JWT认证）
	userGroup := r.Group("/api/user")
	userGroup.Use(auth.AuthMiddleware())
	{
		// 获取当前登录用户资料
		userGroup.GET("/profile", userHandler.GetProfile)
		// 更新当前登录用户资料
		userGroup.PUT("/profile", userHandler.UpdateProfile)
	}
}

// registerRoomRoutes 注册聊天室模块路由（均需JWT认证）：创建、查询、加入、退出、成员列表
func registerRoomRoutes(r *gin.Engine, db *gorm.DB) {
	// 房间模块：仓储 → 服务 → 处理器
	roomRepo := repository.NewRoomRepository(db)
	roomService := service.NewRoomService(roomRepo)
	roomHandler := handler.NewRoomHandler(roomService)

	roomGroup := r.Group("/api/rooms")
	roomGroup.Use(auth.AuthMiddleware())
	{
		// 创建房间
		roomGroup.POST("", roomHandler.CreateRoom)
		// 查询当前用户加入的房间列表
		roomGroup.GET("", roomHandler.GetUserRooms)
		// 查询房间详情
		roomGroup.GET("/:id", roomHandler.GetRoom)
		// 加入房间
		roomGroup.POST("/:id/join", roomHandler.JoinRoom)
		// 退出房间
		roomGroup.POST("/:id/leave", roomHandler.LeaveRoom)
		// 查询房间成员列表
		roomGroup.GET("/:id/members", roomHandler.GetMembers)
	}
}

// registerMessageRoutes 注册消息模块路由（均需JWT认证）：发送、查询、未读数、标记已读
// 消息服务依赖WebSocket Hub做实时推送
func registerMessageRoutes(r *gin.Engine, db *gorm.DB, hub *websocket.Hub) {
	// 消息模块：消息仓储 + 房间仓储（群聊成员权限校验）→ 服务（依赖Hub做WebSocket实时推送）→ 处理器
	messageRepo := repository.NewMessageRepository(db)
	roomRepo := repository.NewRoomRepository(db)
	messageService := service.NewMessageService(messageRepo, hub, roomRepo)
	messageHandler := handler.NewMessageHandler(messageService)

	messageGroup := r.Group("/api/messages")
	messageGroup.Use(auth.AuthMiddleware())
	{
		// 发送消息（私聊传receiver_id，群聊传room_id）
		messageGroup.POST("", messageHandler.SendMessage)
		// 查询消息列表（支持target_id/room_id/page/page_size）
		messageGroup.GET("", messageHandler.GetMessages)
		// 查询未读消息数量
		messageGroup.GET("/unread-count", messageHandler.GetUnreadCount)
		// 标记消息已读
		messageGroup.POST("/:id/read", messageHandler.MarkAsRead)
	}
}

// registerPresenceRoutes 注册在线状态模块路由（均需JWT认证）：在线用户列表、单用户状态查询
// 依赖Redis缓存实时状态与WebSocket Hub广播状态变更
func registerPresenceRoutes(r *gin.Engine, db *gorm.DB, hub *websocket.Hub, redisClient *redis.Client) {
	// 在线状态模块：用户仓储 → 在线状态服务（依赖Redis与Hub）→ 处理器
	userRepo := repository.NewUserRepository(db)
	presenceService := service.NewPresenceService(userRepo, redisClient, hub)
	presenceHandler := handler.NewPresenceHandler(presenceService)

	// 注入在线状态服务：WebSocket连接建立时标记用户在线
	handler.InitPresenceService(presenceService)
	// 注入离线回调：用户最后一个连接断开时标记离线（多开场景下仅最后一个连接断开才触发）
	hub.OnUserDisconnect = func(userID uint) {
		_ = presenceService.SetOffline(userID)
	}

	presenceGroup := r.Group("/api/users")
	presenceGroup.Use(auth.AuthMiddleware())
	{
		// 获取在线用户ID列表
		presenceGroup.GET("/online", presenceHandler.GetOnlineUsers)
		// 查询单个用户在线状态
		presenceGroup.GET("/:id/presence", presenceHandler.GetPresence)
	}
}

// registerNotificationRoutes 注册通知模块路由（均需JWT认证）：查询通知、标记已读
// 通知服务依赖消息仓储（推送离线消息通知）与WebSocket Hub（实时推送），
// 并注入全局通知服务供WebSocket连接建立时触发离线通知推送
func registerNotificationRoutes(r *gin.Engine, db *gorm.DB, hub *websocket.Hub) {
	// 通知模块：通知仓储 + 消息仓储 → 通知服务（依赖Hub实时推送）→ 处理器
	notificationRepo := repository.NewNotificationRepository(db)
	messageRepo := repository.NewMessageRepository(db)
	notificationService := service.NewNotificationService(notificationRepo, messageRepo, hub)
	notificationHandler := handler.NewNotificationHandler(notificationService)

	// 注入全局通知服务：WebSocket连接建立时自动推送离线消息通知
	handler.InitNotificationService(notificationService)

	notificationGroup := r.Group("/api/notifications")
	notificationGroup.Use(auth.AuthMiddleware())
	{
		// 查询当前用户通知列表（支持page/page_size）
		notificationGroup.GET("", notificationHandler.GetNotifications)
		// 标记通知已读
		notificationGroup.POST("/:id/read", notificationHandler.MarkAsRead)
	}
}

// registerUploadRoutes 注册文件上传模块路由：静态文件服务 + 上传接口（需JWT认证）
// 上传的文件保存到本地uploads目录，通过静态服务直接访问，返回URL可用于头像、消息附件等
func registerUploadRoutes(r *gin.Engine, cfg *config.Config) {
	// 静态文件服务：/uploads/* 映射到上传目录（与config中dir、url_prefix对应）
	r.Static(cfg.Upload.URLPrefix, cfg.Upload.Dir)

	// 上传处理器：保存文件并返回可访问URL
	uploadHandler := handler.NewUploadHandler(cfg.Upload.Dir, cfg.Upload.URLPrefix, cfg.Upload.MaxSize)

	uploadGroup := r.Group("/api")
	uploadGroup.Use(auth.AuthMiddleware())
	{
		// 上传文件（表单字段：file）
		uploadGroup.POST("/upload", uploadHandler.Upload)
	}
}

// registerWebRoutes 托管前端静态资源（index.html/js/styles 等）
// 注意：不能使用 r.Static("/", "./web")——其底层注册 catch-all 路由 /*filepath，
// 与已注册的 /api 前缀在 gin 路由树中冲突，启动时会 panic；
// 因此改用 NoRoute 兜底 + http.FileServer（自带路径穿越防护，目录请求自动回退 index.html）
func registerWebRoutes(r *gin.Engine) {
	webFS := http.FileServer(http.Dir("./web"))

	r.NoRoute(func(c *gin.Context) {
		p := c.Request.URL.Path

		// API/WebSocket 路径未命中时返回统一 JSON 404，避免被前端页面兜底误报
		if p == "/api" || strings.HasPrefix(p, "/api/") || p == "/ws" {
			c.JSON(http.StatusNotFound, gin.H{"success": false, "message": "接口不存在"})
			return
		}

		// 前端静态资源仅支持 GET/HEAD
		if c.Request.Method != http.MethodGet && c.Request.Method != http.MethodHead {
			c.Status(http.StatusMethodNotAllowed)
			return
		}

		webFS.ServeHTTP(c.Writer, c.Request)
	})
}
