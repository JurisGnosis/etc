package localstorage

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestInit(t *testing.T) {
	// 创建临时目录用于测试
	tempDir, err := os.MkdirTemp("", "localstorage_test")
	if err != nil {
		t.Fatalf("Failed to create temp directory: %v", err)
	}
	defer os.RemoveAll(tempDir) // 测试结束后清理

	// 测试初始化成功
	err = Init(tempDir, "test-salt")
	if err != nil {
		t.Errorf("Init failed with valid directory: %v", err)
	}

	// 验证全局变量设置正确
	if baseStorageDir != tempDir {
		t.Errorf("baseStorageDir not set correctly, got: %s, want: %s", baseStorageDir, tempDir)
	}
	if usernameSalt != "test-salt" {
		t.Errorf("usernameSalt not set correctly, got: %s, want: %s", usernameSalt, "test-salt")
	}

	// 测试权限不足的情况（仅在非Windows系统上测试）
	if os.Getenv("GOOS") != "windows" {
		readOnlyDir := filepath.Join(tempDir, "readonly")
		if err := os.MkdirAll(readOnlyDir, 0555); err != nil {
			t.Fatalf("Failed to create read-only directory: %v", err)
		}

		// 尝试在只读目录中初始化
		err = Init(readOnlyDir, "test-salt")
		if err == nil {
			t.Error("Init should fail with read-only directory")
		}
	}
}

func TestCalcPath(t *testing.T) {
	// 设置测试环境
	usernameSalt = "test-salt"

	// 测试用例
	testCases := []struct {
		userID   string
		expected int // 期望的路径长度
	}{
		{"user1", 16},
		{"user2", 16},
		{"", 16}, // 空用户ID
	}

	for _, tc := range testCases {
		t.Run(tc.userID, func(t *testing.T) {
			path := CalcPath(tc.userID)
			if len(path) != tc.expected {
				t.Errorf("CalcPath(%q) returned path with length %d, want %d", tc.userID, len(path), tc.expected)
			}

			// 验证路径只包含有效的十六进制字符
			for _, c := range path {
				if !((c >= '0' && c <= '9') || (c >= 'a' && c <= 'f')) {
					t.Errorf("CalcPath(%q) returned invalid hex character %q", tc.userID, c)
					break
				}
			}
		})
	}

	// 验证不同用户ID生成不同的路径
	path1 := CalcPath("user1")
	path2 := CalcPath("user2")
	if path1 == path2 {
		t.Errorf("CalcPath should generate different paths for different users, got %s for both", path1)
	}
}

func TestGetStoragePath(t *testing.T) {
	// 设置测试环境
	tempDir, err := os.MkdirTemp("", "storage_path_test")
	if err != nil {
		t.Fatalf("Failed to create temp directory: %v", err)
	}
	defer os.RemoveAll(tempDir)

	baseStorageDir = tempDir
	usernameSalt = "test-salt"

	// 测试用例
	fileName := "test.txt"
	userID := "testuser"

	path := GetStoragePath(fileName, userID)

	// 验证路径格式
	if !strings.HasPrefix(path, tempDir) {
		t.Errorf("GetStoragePath should start with baseStorageDir, got: %s", path)
	}

	if !strings.HasSuffix(path, fileName) {
		t.Errorf("GetStoragePath should end with fileName, got: %s", path)
	}

	// 验证路径包含用户路径部分
	userPath := CalcPath(userID)
	if !strings.Contains(path, userPath) {
		t.Errorf("GetStoragePath should contain user path %s, got: %s", userPath, path)
	}

	// 验证不同用户生成不同的路径
	path1 := GetStoragePath(fileName, "user1")
	path2 := GetStoragePath(fileName, "user2")
	if path1 == path2 {
		t.Errorf("GetStoragePath should generate different paths for different users")
	}
}

func TestUpload(t *testing.T) {
	// 设置测试环境
	tempDir, err := os.MkdirTemp("", "upload_test")
	if err != nil {
		t.Fatalf("Failed to create temp directory: %v", err)
	}
	defer os.RemoveAll(tempDir)

	baseStorageDir = tempDir
	usernameSalt = "test-salt"

	// 创建测试源文件
	sourceContent := "test content"
	sourceFile := filepath.Join(tempDir, "source.txt")
	if err := os.WriteFile(sourceFile, []byte(sourceContent), 0644); err != nil {
		t.Fatalf("Failed to create source file: %v", err)
	}

	// 测试上传成功
	destPath, err := Upload(sourceFile, "uploaded.txt", "testuser")
	if err != nil {
		t.Errorf("Upload failed: %v", err)
	}

	// 验证目标文件存在
	if _, err := os.Stat(destPath); os.IsNotExist(err) {
		t.Errorf("Destination file does not exist: %s", destPath)
	}

	// 验证文件内容
	destContent, err := os.ReadFile(destPath)
	if err != nil {
		t.Errorf("Failed to read destination file: %v", err)
	}
	if string(destContent) != sourceContent {
		t.Errorf("Destination file content mismatch, got: %s, want: %s", string(destContent), sourceContent)
	}

	// 测试源文件不存在的情况
	_, err = Upload("nonexistent.txt", "uploaded.txt", "testuser")
	if err == nil {
		t.Error("Upload should fail with nonexistent source file")
	}
}

func TestUploadRawContent(t *testing.T) {
	// 设置测试环境
	tempDir, err := os.MkdirTemp("", "upload_raw_test")
	if err != nil {
		t.Fatalf("Failed to create temp directory: %v", err)
	}
	defer os.RemoveAll(tempDir)

	baseStorageDir = tempDir
	usernameSalt = "test-salt"

	// 创建测试源文件
	sourceContent := "raw content test"
	sourceFile := filepath.Join(tempDir, "source_raw.txt")
	if err := os.WriteFile(sourceFile, []byte(sourceContent), 0644); err != nil {
		t.Fatalf("Failed to create source file: %v", err)
	}

	// 测试上传成功
	destPath, err := UploadRawContent(sourceFile, "uploaded_raw.txt", "testuser")
	if err != nil {
		t.Errorf("UploadRawContent failed: %v", err)
	}

	// 验证目标文件存在
	if _, err := os.Stat(destPath); os.IsNotExist(err) {
		t.Errorf("Destination file does not exist: %s", destPath)
	}

	// 验证文件内容
	destContent, err := os.ReadFile(destPath)
	if err != nil {
		t.Errorf("Failed to read destination file: %v", err)
	}
	if string(destContent) != sourceContent {
		t.Errorf("Destination file content mismatch, got: %s, want: %s", string(destContent), sourceContent)
	}

	// 测试源文件不存在的情况
	_, err = UploadRawContent("nonexistent.txt", "uploaded_raw.txt", "testuser")
	if err == nil {
		t.Error("UploadRawContent should fail with nonexistent source file")
	}
}

// 辅助函数：测试目录是否为空
func TestHelperFunctions(t *testing.T) {
	// 测试 CalcPath 的一致性
	usernameSalt = "consistent-salt"
	path1 := CalcPath("user")
	path2 := CalcPath("user")
	if path1 != path2 {
		t.Errorf("CalcPath should be consistent for the same input, got: %s and %s", path1, path2)
	}

	// 测试不同盐值产生不同的路径
	usernameSalt = "salt1"
	path1 = CalcPath("user")
	usernameSalt = "salt2"
	path2 = CalcPath("user")
	if path1 == path2 {
		t.Errorf("CalcPath should generate different paths for different salts")
	}
}
