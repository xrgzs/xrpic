package services

import (
	"crypto/md5"
	"encoding/hex"
	"fmt"
	"io"
	"mime"
	"mime/multipart"
	"net/http"
	"os"
	"path/filepath"
	"strings"
	"time"
	"xrpic/internal/app/config"
	"xrpic/internal/models"

	"github.com/google/uuid"
)

type FileService struct {
	config config.Config
}

func NewFileService(cfg config.Config) *FileService {
	return &FileService{
		config: cfg,
	}
}

// func (fs *FileService) GetUploadDir() string {
// }

// 保存上传的文件
func (fs *FileService) SaveUploadedFile(file *multipart.FileHeader) (models.FullResult, error) {
	// 检查文件大小
	if file.Size > fs.config.Storage.MaxFileSize {
		return models.FullResult{}, fmt.Errorf("file size %d exceeds maximum allowed size of %d bytes", file.Size, fs.config.Storage.MaxFileSize)
	}

	src, err := file.Open()
	if err != nil {
		return models.FullResult{}, err
	}
	defer src.Close()

	// 读取前缀以检测 MIME 类型（最多 512 字节）
	header := make([]byte, 512)
	n, err := src.Read(header)
	if err != nil && err != io.EOF {
		return models.FullResult{}, err
	}
	header = header[:n]

	// 根据前缀检测 MIME
	contentType := http.DetectContentType(header)
	// 拒绝非图片文件上传
	if !strings.HasPrefix(contentType, "image/") {
		return models.FullResult{}, fmt.Errorf("unsupported file type: %s", contentType)
	}

	// 如果没有扩展名，尝试根据内容检测 MIME 类型
	ext := filepath.Ext(file.Filename)
	if ext == "" {
		exts, err := mime.ExtensionsByType(contentType)
		if err == nil && len(exts) > 0 {
			ext = exts[0]
		} else {
			ext = ".jpg" // 默认扩展名
		}
	}

	// 生成日期路径并创建目录
	now := time.Now()
	datePath := now.Format("2006/01")
	dirPath := filepath.Join(config.Conf.Storage.UploadDir, datePath)
	if err := os.MkdirAll(dirPath, 0755); err != nil {
		return models.FullResult{}, err
	}

	// 创建临时文件（和目标目录在同一目录以便原子重命名）
	tmpFile, err := os.CreateTemp(dirPath, "upload-*")
	if err != nil {
		return models.FullResult{}, err
	}
	tmpPath := tmpFile.Name()
	// 确保临时文件在异常情况下被清理
	cleanupTmp := func() {
		tmpFile.Close()
		os.Remove(tmpPath)
	}

	md5Hash := md5.New()
	writer := io.MultiWriter(md5Hash, tmpFile)

	// 先写入已读取的头部
	if len(header) > 0 {
		if _, err := writer.Write(header); err != nil {
			cleanupTmp()
			return models.FullResult{}, err
		}
	}

	// 将剩余内容从 src 流式写入临时文件并同时更新 MD5
	if _, err := io.Copy(writer, src); err != nil {
		cleanupTmp()
		return models.FullResult{}, err
	}

	// 关闭临时文件以便之后重命名
	if err := tmpFile.Close(); err != nil {
		os.Remove(tmpPath)
		return models.FullResult{}, err
	}

	// 计算 MD5 字符串
	md5String := hex.EncodeToString(md5Hash.Sum(nil))

	// 生成最终文件名与路径
	filename := fmt.Sprintf("%s%s", md5String, ext)
	finalPath := filepath.Join(dirPath, filename)

	// 如果目标文件已存在，移除临时文件并直接返回已有 URL
	if info, err := os.Stat(finalPath); err == nil && !info.IsDir() {
		os.Remove(tmpPath)
		return models.FullResult{
			FileName:  filename,
			ImgURL:    fmt.Sprintf("%s/%s/%s", config.Conf.Storage.BaseURL, datePath, filename),
			Extname:   ext,
			Type:      "local",
			ID:        uuid.New(),
			CreatedAt: info.ModTime().Unix(),
			UpdatedAt: time.Now().Unix(),
		}, nil
	}

	// 原子重命名临时文件为最终文件名
	if err := os.Rename(tmpPath, finalPath); err != nil {
		os.Remove(tmpPath)
		return models.FullResult{}, err
	}

	return models.FullResult{
		FileName:  filename,
		ImgURL:    fmt.Sprintf("%s/%s/%s", config.Conf.Storage.BaseURL, datePath, filename),
		Extname:   ext,
		Type:      "local",
		ID:        uuid.New(),
		CreatedAt: time.Now().Unix(),
		UpdatedAt: time.Now().Unix(),
	}, nil
}
