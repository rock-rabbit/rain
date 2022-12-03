package rain

import (
	"context"
	"fmt"
	"math/rand"
	"mime"
	"net/url"
	"os"
	"path/filepath"
	"strings"
	"time"
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
	if kind < 0 {
		kind = 0
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
