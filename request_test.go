package rain

import "testing"

// TestAutoFilename 测试自动获取文件名称
func TestAutoFilename(t *testing.T) {
	data := &resourceInfo{
		uri:                "",
		contentDisposition: "",
		extension:          "",
	}

	if data.getFilename() != "" {
		t.Fatal()
	}
	data.uri = "http://www.68wu.cn/"
	if data.getFilename() != "" {
		t.Fatal()
	}
	data.uri = "http://www.68wu.cn/hello"
	if data.getFilename() != "hello" {
		t.Fatal()
	}
	data.uri = "http://www.68wu.cn/hello?s=x&x=s"
	if data.getFilename() != "hello" {
		t.Fatal()
	}
	data.extension = "jpg"
	if data.getFilename() != "hello.jpg" {
		t.Fatal()
	}
	data.contentDisposition = "attachment;filename=file.xml"
	if data.getFilename() != "file.xml" {
		t.Fatal()
	}
}
