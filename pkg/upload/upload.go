package upload

import (
	"fmt"
	"io"
	"mime/multipart"
	"os"
	"path"
	"path/filepath"
	"time"
)

type UploadService interface {
	SaveFile(file *multipart.FileHeader, subPath string) (string, error)
	ValidateFile(file *multipart.FileHeader, maxSize int64, allowedTypes []string) error
	DeleteFile(filePath string) error
}

type LocalUploadService struct {
	BasePath string
}

func NewLocalUploadService(basePath string) UploadService {
	os.MkdirAll(basePath, 0755)
	return &LocalUploadService{BasePath: basePath}
}

func (s *LocalUploadService) SaveFile(file *multipart.FileHeader, subPath string) (string, error) {
	// 打开文件
	src, err := file.Open()
	if err != nil {
		return "", fmt.Errorf("打开文件失败: %w", err)
	}
	defer src.Close()

	// 生成文件名
	filename := generateFilename(file.Filename)
	fullPath := filepath.Join(s.BasePath, subPath, filename)

	// 确保目录存在
	os.MkdirAll(filepath.Dir(fullPath), 0755)

	// 创建目标文件
	dst, err := os.Create(fullPath)
	if err != nil {
		return "", fmt.Errorf("创建文件失败: %w", err)
	}
	defer dst.Close()

	// 复制文件内容
	if _, err := io.Copy(dst, src); err != nil {
		return "", fmt.Errorf("保存文件失败: %w", err)
	}

	// 返回 URL 风格的相对路径，统一使用正斜杠
	return path.Join(subPath, filename), nil
}

func (s *LocalUploadService) ValidateFile(file *multipart.FileHeader, maxSize int64, allowedTypes []string) error {
	// 检查文件大小
	if file.Size > maxSize {
		return fmt.Errorf("文件大小超过限制: %d bytes", maxSize)
	}

	// 检查文件类型
	contentType := file.Header.Get("Content-Type")
	if !contains(allowedTypes, contentType) {
		return fmt.Errorf("不支持的文件类型: %s", contentType)
	}

	return nil
}

func (s *LocalUploadService) DeleteFile(subpath string) error {
	fullPath := filepath.Join(s.BasePath, subpath)
	return os.Remove(fullPath)
}

func generateFilename(originalName string) string {
	ext := filepath.Ext(originalName)
	timestamp := time.Now().UnixNano()
	randomStr := generateRandomString(8)
	return fmt.Sprintf("%d_%s%s", timestamp, randomStr, ext)
}

func generateRandomString(length int) string {
	// 简单的随机字符串生成
	return fmt.Sprintf("%x", time.Now().UnixNano())[:length]
}

func contains(slice []string, item string) bool {
	for _, s := range slice {
		if s == item {
			return true
		}
	}
	return false
}
