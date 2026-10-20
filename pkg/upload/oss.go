package upload

import (
	"fmt"
	"mime/multipart"

	"github.com/aliyun/aliyun-oss-go-sdk/oss"
)

type OSSUploadService struct {
	client     *oss.Client
	bucket     *oss.Bucket
	endpoint   string
	bucketName string
}

func NewOSSUploadService(endpoint, accessKeyID, accessKeySecret, bucketName string) (UploadService, error) {
	client, err := oss.New(endpoint, accessKeyID, accessKeySecret)
	if err != nil {
		return nil, err
	}

	bucket, err := client.Bucket(bucketName)
	if err != nil {
		return nil, err
	}

	return &OSSUploadService{
		client:     client,
		bucket:     bucket,
		endpoint:   endpoint,
		bucketName: bucketName,
	}, nil
}

func (s *OSSUploadService) SaveFile(file *multipart.FileHeader, subPath string) (string, error) {
	src, err := file.Open()
	if err != nil {
		return "", err
	}
	defer src.Close()

	filename := generateFilename(file.Filename)
	objectKey := fmt.Sprintf("%s/%s", subPath, filename)

	err = s.bucket.PutObject(objectKey, src)
	if err != nil {
		return "", err
	}

	return objectKey, nil
}

func (s *OSSUploadService) ValidateFile(file *multipart.FileHeader, maxSize int64, allowedTypes []string) error {
	if file.Size > maxSize {
		return fmt.Errorf("文件大小超过限制")
	}
	// 其他验证逻辑
	return nil
}

func (s *OSSUploadService) DeleteFile(filepath string) error {
	return s.bucket.DeleteObject(filepath)
}
