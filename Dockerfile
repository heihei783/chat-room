# ============================================================
# chat-app 后端 + 前端一体化镜像（多阶段构建）
# 构建：docker build -t chat-app .
# 说明：
#   - 纯 Go 依赖（gin/gorm/redis/jwt/websocket 等），静态编译，无需 CGO
#   - 前端 web/ 目录直接 COPY 进镜像，由后端 r.Static("/", "./web") 同源托管
#   - MySQL/Redis 为外部服务（config.yaml 的 database/redis 指向远程服务器），
#     镜像内不包含数据库，容器启动时直连外部数据库
# ============================================================

# ---------- 构建阶段 ----------
FROM golang:1.26-alpine AS builder

WORKDIR /build

# 使用国内 Go 模块代理（goproxy.cn，七牛云）加速依赖下载，避免国内服务器访问 GitHub 超时
ENV GOPROXY=https://goproxy.cn,direct

# 先复制依赖清单并下载，充分利用 Docker 层缓存（go.mod/go.sum 不变时跳过下载）
COPY go.mod go.sum ./
RUN go mod download

# 复制全部源码并编译
COPY . .
RUN CGO_ENABLED=0 GOOS=linux go build -trimpath -ldflags="-s -w" -o /out/chat-app ./cmd/server

# ---------- 运行阶段 ----------
FROM alpine:3.20

# 使用阿里云镜像源加速 apk 依赖下载（国内服务器直连官方源超时）；随后安装证书与时区
RUN sed -i 's/dl-cdn.alpinelinux.org/mirrors.aliyun.com/g' /etc/apk/repositories \
    && apk add --no-cache ca-certificates tzdata \
    && cp /usr/share/zoneinfo/Asia/Shanghai /etc/localtime \
    && echo "Asia/Shanghai" > /etc/timezone

WORKDIR /app

# 后端二进制
COPY --from=builder /out/chat-app .

# 配置文件（config 固定名，viper 从 ./config 读取；server.host 已为 0.0.0.0）
COPY config/config.yaml ./config/config.yaml

# 前端静态资源（登录/注册/聊天页面）
COPY web ./web

# 上传目录：运行时建议挂载 volume 持久化（见 docker-compose.yml）
RUN mkdir -p /app/uploads

# 应用端口（与 config.yaml 的 server.port 对应）
EXPOSE 8080

# 启动：加载 ./config/config.yaml，托管 /web 前端与 /uploads 静态文件
CMD ["./chat-app"]
