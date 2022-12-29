package rain

import (
	"context"
	"strings"
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
	var (
		name string
	)
	name = getMimeFilename("attachment;filename=FileName.txt")
	if name != "FileName.txt" {
		t.Fail()
	}
	name = getMimeFilename("filename=FileName.txt")
	if name != "" {
		t.Fail()
	}
	name = getMimeFilename("%%x")
	if name != "" {
		t.Fail()
	}
}

// TestGetUriFilename 测试获取资源链接中的文件名
func TestGetUriFilename(t *testing.T) {
	var (
		name string
	)
	name = getUriFilename("https://www.68wu.com/filename.txt?l=8")
	if name != "filename.txt" {
		t.Fail()
	}
	name = getUriFilename("1")
	if name != "" {
		t.Fail()
	}
}

func TestRandomString(t *testing.T) {
	testData := []struct {
		Size int
		Kind int
	}{
		{5, -1},
		{-1, -1},
		{5, 0},
		{5, 1},
		{5, 2},
		{5, 3},
		{5, 4},
		{20, 1},
	}
	for key, val := range testData {
		s := randomString(val.Size, val.Kind)
		if val.Size >= 0 && len(s) != val.Size {
			t.Fatal(key, "长度错误")
		}
	}
}

func TestContextDone(t *testing.T) {
	ctx, cancel := context.WithCancel(context.Background())
	done := contextDone(ctx)
	if done == true {
		t.Fatal("done 为 true")
	}
	cancel()
	done = contextDone(ctx)
	if done == false {
		t.Fatal("done 为 false")
	}
}

// TestRegexGetOne 获取匹配到的单个字符串
func TestRegexGetOne(t *testing.T) {
	testData := []struct {
		regex   string
		content string
		out     string
	}{
		{``, "", ""},
		{`数量：(\d+)`, "---数量：827", "827"},
		{`数量：\d+`, "---数量：827", ""},
		{`数量：\d+---(\d+)`, "---数量：827---928---", "928"},
	}
	for _, v := range testData {
		tmp := regexGetOne(v.regex, v.content)
		if tmp != v.out {
			t.Errorf("匹配单个字符串失败, 输出 %s, 应输出 %s", tmp, v.out)
		}
	}
}

// TestFormatFileSize 测试字节的单位转换
func TestFormatFileSize(t *testing.T) {
	testData := []struct {
		size       int64
		formatSize string
	}{
		{-1, "0.00 B"},
		{0, "0.00 B"},
		{1, "1.00 B"},
		{627, "627.00 B"},
		{1024, "1.00 Kib"},
		{1025, "1.00 Kib"},
		{2042, "1.99 Kib"},
		{1048576, "1.00 Mib"},
		{1073741824, "1.00 Gib"},
		{1099511627776, "1.00 Tib"},
		{1.1259e+15, "1.00 Eib"},
	}

	for _, val := range testData {
		tmp := formatFileSize(val.size)
		if tmp != val.formatSize {
			t.Errorf("%v 测试失败, 输出为: %s 应该为: %v\n", val, tmp, val.formatSize)
		}
	}

}

// TestFilterFileName windows 规定过滤掉非法字符
func TestFilterFileName(t *testing.T) {
	names := [][2]string{
		{"这在济南是合法的?", "这在济南是合法的"},
		{"大明湖\\红叶谷\\千佛山，这些都很好玩", "大明湖红叶谷千佛山，这些都很好玩"},
		{"最好玩的地方是融创乐园/融创乐园", "最好玩的地方是融创乐园融创乐园"},
		{"哔哔***哔哔", "哔哔哔哔"},
		{"宝贝\"我爱你\"", "宝贝我爱你"},
		{"这里输入<文件名>", "这里输入文件名"},
		{"济南:山东的省会", "济南山东的省会"},
		{strings.Repeat("1", 256), strings.Repeat("1", 255)},
	}

	for _, v := range names {
		tmp := filterFileName(v[0])
		if tmp != v[1] {
			t.Errorf("过滤掉非法字符失败, 输出 %s, 应输出 %s", tmp, v[1])
		}
	}
}
