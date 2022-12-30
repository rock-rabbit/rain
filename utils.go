package rain

import (
	"bytes"
	"context"
	"fmt"
	"io"
	"math/rand"
	"mime"
	"net/url"
	"os"
	"path/filepath"
	"regexp"
	"strings"
	"time"
	"unicode/utf8"
)

// contextDone context 是否已经完成
func contextDone(ctx context.Context) bool {
	select {
	case <-ctx.Done():
		return true
	default:
		return false
	}
}

// fileExist 文件是否存在
func fileExist(path string) bool {
	_, err := os.Stat(path)
	if err != nil {
		if os.IsNotExist(err) {
			return false
		}
	}
	return true
}

// getFilename 获取附加的文件名称
func getMimeFilename(s string) string {
	_, params, err := mime.ParseMediaType(s)
	if err != nil {
		return ""
	}
	if val, ok := params["filename"]; ok {
		return val
	}
	return ""
}

// getUriFilename 获取资源链接中的文件名
func getUriFilename(s string) string {
	u, _ := url.Parse(s)
	if u != nil {
		us := strings.Split(u.Path, "/")
		if len(us) > 1 {
			return us[len(us)-1]
		}
	}
	return ""
}

// randomString 随机数
// size 随机码的位数
// kind 0=纯数字,1=小写字母,2=大写字母,3=数字、大小写字母
func randomString(size int, kind int) string {
	if size < 1 {
		return ""
	}
	ikind, kinds, rsbytes := kind, [][]int{{10, 48}, {26, 97}, {26, 65}}, make([]byte, size)
	isAll := kind > 2 || kind < 0
	rand.Seed(time.Now().UnixNano())
	for i := 0; i < size; i++ {
		if isAll { // random ikind
			ikind = rand.Intn(3)
		}
		scope, base := kinds[ikind][0], kinds[ikind][1]
		rsbytes[i] = uint8(base + rand.Intn(scope))
	}
	return string(rsbytes)
}

// formatFileSize 字节的单位转换 保留两位小数
func formatFileSize(fileSize int64) string {
	var sizef = float64(fileSize)
	if sizef <= 0 {
		return "0.00 B"
	}
	if sizef < 1024 {
		return fmt.Sprintf("%.2f B", sizef/float64(1))
	} else if sizef < 1048576 {
		return fmt.Sprintf("%.2f Kib", sizef/float64(1024))
	} else if sizef < 1073741824 {
		return fmt.Sprintf("%.2f Mib", sizef/float64(1048576))
	} else if sizef < 1099511627776 {
		return fmt.Sprintf("%.2f Gib", sizef/float64(1073741824))
	} else if sizef < 1125899906842624 {
		return fmt.Sprintf("%.2f Tib", sizef/float64(1099511627776))
	} else {
		return fmt.Sprintf("%.2f Eib", sizef/float64(1125899906842624))
	}
}

// autoFileRenaming 自动文件重命名，寻找不冲突的命名
func autoFileRenaming(dir, name string) (string, string) {
	i := 1
	ext := filepath.Ext(name)
	name = strings.TrimSuffix(name, ext)
	var path string
	var filename string
	for {
		filename = fmt.Sprintf("%s.%d%s", name, i, ext)
		path = filepath.Join(dir, filename)
		if !fileExist(path) {
			break
		}
		i++
	}
	return path, filename
}

// filterFileName 过滤非法字符
func filterFileName(name string) string {
	// 过滤头部的空格
	name = strings.TrimPrefix(name, regexGetOne(`^([[:blank:]]+)`, name))
	// 过滤非法字符
	for _, v := range []rune{'?', '\\', '/', '*', '"', '<', '>', '|', ':'} {
		name = strings.ReplaceAll(name, string(v), "")
	}
	// 截取前 255 个字
	if getStringLength(name) > 255 {
		i := 0
		c := bytes.NewBufferString("")
		for _, v := range name {
			c.WriteString(string(v))
			i++
			if i == 255 {
				break
			}
		}
		name = c.String()
	}
	return name
}

// getStringLength 获取字符串的长度
func getStringLength(str string) int {
	return utf8.RuneCountInString(str)
}

// regexGetOne 获取匹配到的单个字符串
func regexGetOne(str, s string) string {
	re := regexp.MustCompile(str)
	submatch := re.FindStringSubmatch(s)
	if len(submatch) <= 1 {
		return ""
	}
	return submatch[1]
}

// writeFunc 代理写入
type writeFunc struct {
	writeFunc func([]byte) (n int, err error)
}

// newWriteFunc 代理写入
func newWriteFunc(w func([]byte) (n int, err error)) io.Writer {
	return &writeFunc{
		writeFunc: w,
	}
}

// Write 文件写入
func (fa *writeFunc) Write(p []byte) (n int, err error) {
	return fa.writeFunc(p)
}

// MultiReadable 重复读取 reader
type MultiReadable struct {
	originReader io.Reader
	reader       io.Reader
	cache        *bytes.Buffer
}

func NewMultiReadable(reader io.Reader) *MultiReadable {
	return &MultiReadable{
		originReader: reader,
		reader:       reader,
	}
}

func (mr *MultiReadable) Read(p []byte) (int, error) {
	n, err := mr.reader.Read(p)
	// 如果 reader 不支持Seek，则把读取出来的内容同时写入到一个buffer中
	if _, ok := mr.reader.(io.Seeker); !ok && n > 0 {
		if mr.cache == nil {
			mr.cache = &bytes.Buffer{}
		}
		mr.cache.Write(p[:n])
	}
	return n, err
}

func (mr *MultiReadable) Reset() error {
	// 如果reader支持Seek，直接使用Seek即可
	if seeker, ok := mr.reader.(io.Seeker); ok {
		_, err := seeker.Seek(0, io.SeekStart)
		return err
	}
	if mr.cache != nil && mr.cache.Len() > 0 {
		mr.reader = io.MultiReader(mr.cache, mr.reader)
		mr.cache = nil
	}
	return nil
}

func (mr *MultiReadable) Close() error {
	if closer, ok := mr.originReader.(io.Closer); ok {
		return closer.Close()
	}
	return nil
}
