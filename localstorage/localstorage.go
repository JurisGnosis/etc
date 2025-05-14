package localstorage

import (
	"crypto/sha1"
	"encoding/hex"
	"errors"
	"fmt"
	"io"
	"log/slog"
	"os"
	"path/filepath"
	"strings"
	"time"
)

// 存储文件的根目录
var baseStorageDir string

// 与用户路径计算相关，建议不要修改
var usernameSalt string

// 前端访问文件的公共路径
var frontendPublicPath string

// 用于检查文件系统权限的测试文件名
var testFileName = "access_check.tmp"

// Init initializes the local filesystem storage with the provided base directory and settings
// publicPathPrefix is the public URL path that maps to storageDir, used for constructing URLs in upload methods
func Init(storageDir string, sha1Key string, publicPathPrefix string) (err error) {
	slog.Info("initializing local filesystem storage", "directory", storageDir, "publicPathPrefix", publicPathPrefix)
	// 验证存储目录是否存在，如果不存在则创建
	if _, err = os.Stat(storageDir); os.IsNotExist(err) {
		if err = os.MkdirAll(storageDir, 0777); err != nil {
			slog.Error("failed to create storage directory", "error", err)
			return
		}
	}

	// 检查目录的读写权限
	testFilePath := filepath.Join(storageDir, testFileName)

	// 尝试写入测试文件
	testContent := time.Now().String()
	if err = os.WriteFile(testFilePath, []byte(testContent), 0777); err != nil {
		slog.Error("write permission check failed", "error", err)
		return errors.New("storage directory write permission check failed")
	}

	// 尝试读取测试文件
	readContent, err := os.ReadFile(testFilePath)
	if err != nil {
		slog.Error("read permission check failed", "error", err)
		return errors.New("storage directory read permission check failed")
	}

	// 验证内容是否一致
	if string(readContent) != testContent {
		slog.Error("content verification failed", "expected", testContent, "actual", string(readContent))
		return errors.New("storage directory content verification failed")
	}

	// 清理测试文件
	if err = os.Remove(testFilePath); err != nil {
		slog.Warn("failed to clean up test file", "error", err)
		// 不返回错误，因为这不是致命问题
	}

	// 设置全局变量
	baseStorageDir = storageDir
	usernameSalt = sha1Key

	// 确保URL格式正确
	// 1. 确保协议后有双斜杠
	if strings.HasPrefix(publicPathPrefix, "http:") && !strings.HasPrefix(publicPathPrefix, "http://") {
		publicPathPrefix = "http://" + strings.TrimPrefix(publicPathPrefix, "http:/")
	} else if strings.HasPrefix(publicPathPrefix, "https:") && !strings.HasPrefix(publicPathPrefix, "https://") {
		publicPathPrefix = "https://" + strings.TrimPrefix(publicPathPrefix, "https:/")
	}

	// 2. 确保末尾没有斜杠
	if strings.HasSuffix(publicPathPrefix, "/") {
		frontendPublicPath = strings.TrimSuffix(publicPathPrefix, "/")
	} else {
		frontendPublicPath = publicPathPrefix
	}

	slog.Info("local filesystem storage initialized successfully", "directory", storageDir)
	return nil
}

// CalcPath generates a path based on the user identifier and salt
func CalcPath(userIdentifier string) string {
	hasher := sha1.New()
	hasher.Write([]byte(userIdentifier + usernameSalt))
	return hex.EncodeToString(hasher.Sum(nil))[:16]
}

// GetStoragePath generates the full storage path for a file
func GetStoragePath(targetFileName string, userIdentifier string) string {
	userPath := CalcPath(userIdentifier)
	timePath := CalcPath(time.Now().String())[:8]
	return filepath.Join(baseStorageDir, userPath, timePath, targetFileName)
}

// Upload copies a file from a source path to the storage location
func Upload(localFilePath string, targetFileName string, userIdentifier string) (filePath string, err error) {
	slog.Info("uploading file", "localFilePath", localFilePath, "targetFileName", targetFileName, "userIdentifier", userIdentifier, "frontendPublicPath", frontendPublicPath)
	// 检查源文件是否存在
	if _, err = os.Stat(localFilePath); os.IsNotExist(err) {
		slog.Error("source file does not exist", "path", localFilePath)
		return "", errors.New("source file does not exist")
	}

	// 生成目标路径
	destPath := GetStoragePath(targetFileName, userIdentifier)
	destDir := filepath.Dir(destPath)

	// 确保目标目录存在
	if err = os.MkdirAll(destDir, 0777); err != nil {
		slog.Error("failed to create destination directory", "directory", destDir, "error", err)
		return "", fmt.Errorf("failed to create destination directory: %w", err)
	}

	// 打开源文件
	srcFile, err := os.Open(localFilePath)
	if err != nil {
		slog.Error("failed to open source file", "path", localFilePath, "error", err)
		return "", fmt.Errorf("failed to open source file: %w", err)
	}
	defer srcFile.Close()

	// 创建目标文件
	destFile, err := os.Create(destPath)
	if err != nil {
		slog.Error("failed to create destination file", "path", destPath, "error", err)
		return "", fmt.Errorf("failed to create destination file: %w", err)
	}
	defer destFile.Close()

	// 设置文件权限为777
	if err = os.Chmod(destPath, 0777); err != nil {
		slog.Error("failed to set file permissions", "path", destPath, "error", err)
		return "", fmt.Errorf("failed to set file permissions: %w", err)
	}

	// 复制文件内容
	if _, err = io.Copy(destFile, srcFile); err != nil {
		slog.Error("failed to copy file content", "error", err)
		return "", fmt.Errorf("failed to copy file content: %w", err)
	}

	// 构建前端访问URL
	relativePath := destPath[len(baseStorageDir):]
	var publicURL string
	relativePath = filepath.ToSlash(relativePath)
	if strings.HasPrefix(relativePath, "/") {
		publicURL = frontendPublicPath + relativePath
	} else {
		publicURL = frontendPublicPath + "/" + relativePath
	}

	slog.Info("file uploaded successfully", "source", localFilePath, "destination", destPath, "publicURL", publicURL)
	return publicURL, nil
}

// UploadRawContent writes string content to a file in the storage location
func UploadRawContent(plainText string, targetFileName string, userIdentifier string) (filePath string, err error) {
	slog.Info("uploading raw content", "targetFileName", targetFileName, "userIdentifier", userIdentifier, "frontendPublicPath", frontendPublicPath)
	// 生成目标路径
	destPath := GetStoragePath(targetFileName, userIdentifier)
	destDir := filepath.Dir(destPath)

	// 确保目标目录存在
	if err = os.MkdirAll(destDir, 0777); err != nil {
		slog.Error("failed to create destination directory", "directory", destDir, "error", err)
		return "", fmt.Errorf("failed to create destination directory: %w", err)
	}

	// 写入内容到文件
	if err = os.WriteFile(destPath, []byte(plainText), 0777); err != nil {
		slog.Error("failed to write content to file", "path", destPath, "error", err)
		return "", fmt.Errorf("failed to write content to file: %w", err)
	}

	// 构建前端访问URL
	relativePath := destPath[len(baseStorageDir):]
	relativePath = filepath.ToSlash(relativePath)
	var publicURL string
	if strings.HasPrefix(relativePath, "/") {
		publicURL = frontendPublicPath + relativePath
	} else {
		publicURL = frontendPublicPath + "/" + relativePath
	}

	slog.Info("content uploaded successfully", "destination", destPath, "publicURL", publicURL)
	return publicURL, nil
}

// UploadRawBytes writes bytes content to a file in the storage location
func UploadRawBytes(content []byte, targetFileName string, userIdentifier string) (filePath string, err error) {
	slog.Info("uploading raw bytes", "targetFileName", targetFileName, "userIdentifier", userIdentifier, "frontendPublicPath", frontendPublicPath)
	// 生成目标路径
	destPath := GetStoragePath(targetFileName, userIdentifier)
	destDir := filepath.Dir(destPath)

	// 确保目标目录存在
	if err = os.MkdirAll(destDir, 0777); err != nil {
		slog.Error("failed to create destination directory", "directory", destDir, "error", err)
		return "", fmt.Errorf("failed to create destination directory: %w", err)
	}

	// 写入内容到文件
	if err = os.WriteFile(destPath, content, 0777); err != nil {
		slog.Error("failed to write content to file", "path", destPath, "error", err)
		return "", fmt.Errorf("failed to write content to file: %w", err)
	}

	// 构建前端访问URL
	relativePath := destPath[len(baseStorageDir):]
	relativePath = filepath.ToSlash(relativePath)
	var publicURL string
	if strings.HasPrefix(relativePath, "/") {
		publicURL = frontendPublicPath + relativePath
	} else {
		publicURL = frontendPublicPath + "/" + relativePath
	}

	return publicURL, nil
}
