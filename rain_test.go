package rain_test

import (
	"crypto/md5"
	"fmt"
	"io"
	"net"
	"net/http"
	"net/http/httptest"
	"net/url"
	"os"
	"path/filepath"
	"regexp"
	"strconv"
	"strings"
	"testing"
	"time"

	"github.com/rock-rabbit/rain"
)

func init() {
	// 删除临时文件
	os.RemoveAll("./tmp")
	// 测试配置
	rain.SetOutdir("./tmp")
}

// NewFileServer 新建测试文件服务
func NewFileServer(t *testing.T, path string, exec func(w http.ResponseWriter, r *http.Request)) *httptest.Server {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		data, err := os.ReadFile(path)
		if err != nil {
			t.Fatal(err)
		}

		w.Header().Set("etag", MD5(data))
		w.Header().Set("content-length", fmt.Sprint(len(data)))
		w.Header().Set("accept-ranges", "bytes")

		hrange := r.Header.Get("range")
		if hrange != "" {
			ranges := regexp.MustCompile(`bytes=(\d+)-(\d+)`).FindStringSubmatch(hrange)
			if len(ranges) != 3 {
				t.Fatal("bytes 长度错误")
			}
			start, err := strconv.ParseInt(ranges[1], 10, 64)
			if err != nil {
				t.Fatal(err)
			}
			end, err := strconv.ParseInt(ranges[2], 10, 64)
			if err != nil {
				t.Fatal(err)
			}
			w.Header().Set("content-range", fmt.Sprintf("bytes %d-%d/%d", start, end, len(data)))
			w.Header().Set("content-length", fmt.Sprint(end-start+1))

			exec(w, r)

			w.WriteHeader(http.StatusPartialContent)
			w.Write(data[start:end])
			return
		}

		exec(w, r)

		w.WriteHeader(http.StatusOK)
		w.Write(data)
	}))
	_, filename := filepath.Split(path)
	server.URL = server.URL + "/test/" + filename
	return server
}

func MD5(data []byte) string {
	return fmt.Sprintf("%x", md5.Sum(data))
}

func FileMD5(path string) string {
	data, _ := os.ReadFile(path)
	return MD5(data)
}

type TestFile struct {
	Path string
	Name string
	MD5  string
}

var tf = []*TestFile{
	{
		Path: "./test/test_720p_5m.mp4",
		Name: "test_720p_5m.mp4",
		MD5:  "7e245fc2483742414604ce7e67c13111",
	},
}

// Init 初始化
func Init() {
	os.RemoveAll("./tmp")
}

// TestSingleThread 测试单协程下载
func TestSingleThread(t *testing.T) {
	Init()
	for key, val := range tf {
		server := NewFileServer(t, val.Path, func(w http.ResponseWriter, r *http.Request) {})
		ctl := rain.New(server.URL).Run()
		if ctl.Error() != nil {
			t.Fatal(ctl.Error())
		}
		if FileMD5(ctl.Outpath()) != val.MD5 {
			t.Fatal(key, "md5 错误")
		}
	}
}

// TestMultithread 测试多协程下载
func TestMultithread(t *testing.T) {
	Init()
	for key, val := range tf {
		server := NewFileServer(t, val.Path, func(w http.ResponseWriter, r *http.Request) {})
		ctl := rain.New(server.URL, rain.WithRoutineCount(3)).Run()
		if ctl.Error() != nil {
			t.Fatal(ctl.Error())
		}
		if FileMD5(ctl.Outpath()) != val.MD5 {
			t.Fatal(key, "md5 错误")
		}
	}
}

// TestBar 测试 Bar 进度条
func TestBar(t *testing.T) {
	Init()
	for key, val := range tf {
		server := NewFileServer(t, val.Path, func(w http.ResponseWriter, r *http.Request) {})
		ctl := rain.New(server.URL, rain.WithBar()).Run()
		if ctl.Error() != nil {
			t.Fatal(ctl.Error())
		}
		if FileMD5(ctl.Outpath()) != val.MD5 {
			t.Fatal(key, "md5 错误")
		}
	}
}

// TestClose 测试取消下载
func TestClose(t *testing.T) {
	Init()
	for key, val := range tf {
		server := NewFileServer(t, val.Path, func(w http.ResponseWriter, r *http.Request) {})
		ctl := rain.New(server.URL, rain.WithSpeedLimit(1024<<10))
		go func() {
			time.Sleep(time.Second)
			ctl.Close()
		}()
		ctl.Run()
		if ctl.Error() != nil {
			t.Fatal(ctl.Error())
		}
		// 检查文件是否下载完成
		if FileMD5(ctl.Outpath()) == val.MD5 {
			t.Fatal(key, "不应该下载完成")
		}
		// 检查断点文件
		_, err := os.Stat(ctl.Outpath() + ".temp.rain")
		if os.IsNotExist(err) {
			t.Fatal(key, "断点文件不存在")
		}
	}
}

// TestAutoFileRenaming 测试文件重命名
func TestAutoFileRenaming(t *testing.T) {
	Init()
	for key, val := range tf {
		server := NewFileServer(t, val.Path, func(w http.ResponseWriter, r *http.Request) {})

		// 创建同名文件
		os.Mkdir("./tmp", os.ModePerm)
		f, err := os.Create("./tmp/" + val.Name)
		if err != nil {
			t.Fatal(key, err)
		}
		f.Close()

		ctl := rain.New(server.URL, rain.WithAutoFileRenaming(true)).Run()
		if ctl.Error() != nil {
			t.Fatal(ctl.Error())
		}

		// 检查文件名称
		ext := filepath.Ext(val.Path)
		filename := fmt.Sprintf("%s.%d%s", strings.TrimSuffix(val.Name, ext), 1, ext)
		_, outname := filepath.Split(ctl.Outpath())
		if filename != outname {
			t.Fatal(key, "文件名称错误", filename, outname)
		}
	}
}

// TcpServer tcp 测试服务
func TcpServer(t *testing.T, c func(conn net.Conn)) net.Listener {
	l, err := net.Listen("tcp", "127.0.0.1:0")
	if err != nil {
		t.Fatal(err)
	}

	go func() {
		for {
			conn, err := l.Accept()
			if err != nil {
				return
			}
			go c(conn)
		}
	}()
	return l
}

// TestProxy 测试代理
func TestProxy(t *testing.T) {
	Init()
	defer func() {
		rain.SetProxy(http.ProxyFromEnvironment)
	}()

	proxyurl := ""

	l := TcpServer(t, func(conn net.Conn) {
		defer conn.Close()

		b := make([]byte, 1024)
		n, _ := conn.Read(b)

		regurl := regexp.MustCompile(`GET (.*?) HTTP/1`).FindStringSubmatch(string(b))
		if len(regurl) <= 1 {
			return
		}
		urlstr, err := url.Parse(regurl[1])
		if err != nil {
			fmt.Println(err)
		}

		proxyurl = regurl[1]

		sconn, err := net.Dial("tcp", urlstr.Host)
		if err != nil {
			fmt.Println(err)
		}

		_, err = sconn.Write(b[:n])
		if err != nil {
			fmt.Println(err)
		}

		go io.Copy(sconn, conn)
		io.Copy(conn, sconn)

	})
	defer l.Close()
	proxy := l.Addr().String()

	for key, val := range tf {
		server := NewFileServer(t, val.Path, func(w http.ResponseWriter, r *http.Request) {})
		u, err := url.Parse("http://" + proxy)
		if err != nil {
			t.Fatal(err)
		}
		rain.SetProxy(http.ProxyURL(u))
		ctl := rain.New(server.URL).Run()
		if ctl.Error() != nil {
			t.Fatal(key, ctl.Error())
		}
		// 是否经过代理
		if proxyurl != server.URL {
			t.Fatal(key, "proxyurl != server.URL")
		}
		// 检查文件是否下载完成
		if FileMD5(ctl.Outpath()) != val.MD5 {
			t.Fatal(key, "md5 错误")
		}
	}
}
