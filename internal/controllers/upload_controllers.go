package controllers

import (
	"fmt"
	"net/http"
	"os"
	"path/filepath"
	"strings"

	"github.com/gin-gonic/gin"

	"xrpic/internal/app/config"
	"xrpic/internal/models"
	"xrpic/internal/services"
)

type UploadController struct {
	fileService *services.FileService
}

func NewUploadController(fileService *services.FileService) *UploadController {
	return &UploadController{
		fileService: fileService,
	}
}

// 上传处理器
func (uc *UploadController) Upload(c *gin.Context) {
	contentType := c.GetHeader("Content-Type")

	// 1. 表单上传 (multipart/form-data)
	if strings.Contains(contentType, "multipart/form-data") {
		uc.handleFormUpload(c)
		return
	}

	// 2. JSON 请求
	if strings.Contains(contentType, "application/json") {
		var req models.UploadRequest
		if err := c.ShouldBindJSON(&req); err != nil {
			c.JSON(http.StatusBadRequest, models.UploadResponse{
				Success: false,
				Message: "Invalid JSON request",
			})
			return
		}

		// 如果 list 为空,上传剪贴板图片
		if len(req.List) == 0 {
			c.JSON(http.StatusNotImplemented, models.UploadResponse{
				Success: false,
				Message: "Clipboard upload not supported in this implementation",
			})
			return
		}

		// 上传指定路径的图片
		c.JSON(http.StatusNotImplemented, models.UploadResponse{
			Success: false,
			Message: "Path upload not supported in this implementation",
		})
		return
	}

	c.JSON(http.StatusBadRequest, models.UploadResponse{
		Success: false,
		Message: "Unsupported content type",
	})
}

// 处理表单上传
func (uc *UploadController) handleFormUpload(c *gin.Context) {
	form, err := c.MultipartForm()
	if err != nil {
		c.JSON(http.StatusBadRequest, models.UploadResponse{
			Success: false,
			Message: fmt.Sprintf("Failed to parse form: %v", err),
		})
		return
	}

	var urls []string
	var results []models.FullResult

	// 遍历所有文件字段
	for _, files := range form.File {
		for _, file := range files {
			result, err := uc.fileService.SaveUploadedFile(file)
			if err != nil {
				// 检查是否是文件大小超限错误
				if strings.Contains(err.Error(), "exceeds maximum allowed size") {
					c.JSON(http.StatusRequestEntityTooLarge, models.UploadResponse{
						Success: false,
						Message: "File size exceeds maximum allowed size",
					})
					return
				}
				c.JSON(http.StatusInternalServerError, models.UploadResponse{
					Success: false,
					Message: fmt.Sprintf("Failed to save file: %v", err),
				})
				return
			}
			urls = append(urls, result.ImgURL)
			results = append(results, result)
		}
	}

	if len(urls) == 0 {
		c.JSON(http.StatusBadRequest, models.UploadResponse{
			Success: false,
			Message: "No files uploaded",
		})
		return
	}

	c.JSON(http.StatusOK, models.UploadResponse{
		Success:    true,
		Result:     urls,
		FullResult: results,
	})
}

// 删除处理器
func (uc *UploadController) Delete(c *gin.Context) {
	var req models.DeleteRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, models.UploadResponse{
			Success: false,
			Message: "Invalid JSON request",
		})
		return
	}

	if len(req.List) == 0 {
		c.JSON(http.StatusBadRequest, models.UploadResponse{
			Success: false,
			Message: "No files to delete",
		})
		return
	}

	// 遍历删除文件
	for _, item := range req.List {
		if item.Type != "local" {
			c.JSON(http.StatusBadRequest, models.UploadResponse{
				Success: false,
				Message: fmt.Sprintf("Unsupported file type for deletion: %s - %s", item.Type, item.ImgURL),
			})
			return
		}
		// 解析文件路径
		relativePath := strings.TrimPrefix(item.ImgURL, config.Conf.Storage.BaseURL+"/")
		filePath := filepath.Join(config.Conf.Storage.UploadDir, relativePath)

		// 检测文件是否存在
		info, err := os.Stat(filePath)
		if err != nil {
			if os.IsNotExist(err) {
				c.JSON(http.StatusNotFound, models.UploadResponse{
					Success: false,
					Message: fmt.Sprintf("File not found: %s", item.ImgURL),
				})
				return
			}
			// 其他错误（如权限拒绝）返回500
			c.JSON(http.StatusInternalServerError, models.UploadResponse{
				Success: false,
				Message: fmt.Sprintf("Failed to access file: %v", err),
			})
			return
		}
		if info.IsDir() {
			c.JSON(http.StatusBadRequest, models.UploadResponse{
				Success: false,
				Message: fmt.Sprintf("Target is a directory: %s", item.ImgURL),
			})
			return
		}

		// 删除文件
		if err := os.Remove(filePath); err != nil {
			c.JSON(http.StatusInternalServerError, models.UploadResponse{
				Success: false,
				Message: fmt.Sprintf("Failed to delete file: %v", err),
			})
			return
		}
	}

	c.JSON(http.StatusOK, models.UploadResponse{
		Success: true,
		Message: "删除成功",
	})
}
