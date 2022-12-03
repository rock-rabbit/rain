package rain

import (
	"testing"
)

// TestFileExist 测试文件是否存在
func TestFileExist(t *testing.T) {
	path := "./go.mod"
	if !fileExist(path) {
		t.Fatal(path, "文件应该存在")
	}
}

// TestGetMimeFilename 测试获取附加文件名
func TestGetMimeFilename(t *testing.T) {
	name := getMimeFilename("attachment;filename=FileName.txt")
	if name != "FileName.txt" {
		t.Fail()
	}
}

// TestGetUriFilename 测试获取资源链接中的文件名
func TestGetUriFilename(t *testing.T) {
	uri := "https://www.68wu.com/filename.txt?l=8"
	name := getUriFilename(uri)
	if name != "filename.txt" {
		t.Fail()
	}
}
