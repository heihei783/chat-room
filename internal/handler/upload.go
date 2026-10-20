package handler

import (
	"fmt"
	"math/rand"
	"net/http"
	"os"
	"path/filepath"
	"strings"
	"time"

	"github.com/gin-gonic/gin"
)

// allowedExts 允许上传的文件扩展名白名单（头像与消息附件常用类型）
var allowedExts = map[string]bool{
	".jpg": true, ".jpeg": true, ".png": true, ".gif": true, ".webp": true, ".svg": true, // 图片
	".mp4": true, ".mov": true, ".webm": true, // 视频
	".mp3": true, ".wav": true, ".aac": true, // 音频
	".pdf": true, // 文档
	".doc": true, ".docx": true, ".xls": true, ".xlsx": true, ".ppt": true, ".pptx": true, // Office
	".txt": true, ".zip": true, ".rar": true, ".7z": true, // 其他
}

// UploadHandler 文件上传HTTP处理器：接收文件并保存到本地，返回可访问的URL
type UploadHandler struct {
	dir       string // 文件存储目录
	urlPrefix string // 静态访问URL前缀
	maxSize   int64  // 单文件大小上限（字节）
}

// NewUploadHandler 创建文件上传处理器
func NewUploadHandler(dir, urlPrefix string, maxSize int64) *UploadHandler {
	return &UploadHandler{dir: dir, urlPrefix: urlPrefix, maxSize: maxSize}
}

// Upload 处理文件上传请求：POST /api/upload（需JWT认证）
// 表单字段：file（必填）；返回文件的访问URL，可用于头像、消息附件等
func (h *UploadHandler) Upload(c *gin.Context) {
	// 1. 读取上传文件（限制内存占用，超过的部分落临时文件）
	file, err := c.FormFile("file")
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{
			"success": false,
			"message": "请提供要上传的文件（字段名：file）",
		})
		return
	}

	// 2. 校验文件大小（超过上限直接拒绝）
	if file.Size > h.maxSize {
		c.JSON(http.StatusBadRequest, gin.H{
			"success": false,
			"message": fmt.Sprintf("文件大小超过限制（最大%dMB）", h.maxSize/1024/1024),
		})
		return
	}

	// 3. 校验文件扩展名（白名单，防止上传可执行等危险类型）
	ext := strings.ToLower(filepath.Ext(file.Filename))
	if !allowedExts[ext] {
		c.JSON(http.StatusBadRequest, gin.H{
			"success": false,
			"message": "不支持的文件类型",
		})
		return
	}

	// 4. 按日期生成存储子目录（如 uploads/2026/10/15/），避免单目录文件过多
	dateDir := time.Now().Format("2006/01/02")
	saveDir := filepath.Join(h.dir, dateDir)
	if err := os.MkdirAll(saveDir, 0755); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{
			"success": false,
			"message": "创建存储目录失败",
		})
		return
	}

	// 5. 生成唯一文件名（时间戳+随机数，不使用原始文件名，避免路径注入与重名覆盖）
	fileName := fmt.Sprintf("%d_%d%s", time.Now().UnixNano(), rand.Intn(100000), ext)
	savePath := filepath.Join(saveDir, fileName)

	// 6. 保存文件到磁盘
	if err := c.SaveUploadedFile(file, savePath); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{
			"success": false,
			"message": "文件保存失败",
		})
		return
	}

	// 7. 返回文件访问URL（相对路径，前端拼接服务器地址即可访问）
	url := fmt.Sprintf("%s/%s/%s", strings.TrimSuffix(h.urlPrefix, "/"), dateDir, fileName)
	c.JSON(http.StatusOK, gin.H{
		"success": true,
		"message": "上传成功",
		"data": gin.H{
			"url":  url,
			"name": file.Filename,
			"size": file.Size,
		},
	})
}
