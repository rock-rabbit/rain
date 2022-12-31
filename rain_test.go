package rain_test

import (
	"bytes"
	"context"
	"crypto/md5"
	"errors"
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
func NewFileServer(t *testing.T, path string, exec ...func(w http.ResponseWriter, r *http.Request)) *httptest.Server {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		Execute := func(w http.ResponseWriter, r *http.Request) {
			if len(exec) > 0 {
				exec[0](w, r)
			}
		}

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

			Execute(w, r)

			w.WriteHeader(http.StatusPartialContent)
			io.Copy(w, bytes.NewBuffer(data[start:end+1]))
			return
		}

		Execute(w, r)

		w.WriteHeader(http.StatusOK)
		io.Copy(w, bytes.NewBuffer(data))
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
		server := NewFileServer(t, val.Path)
		ctl, err := rain.New(server.URL, rain.WithBar()).Run()
		if err != nil {
			t.Fatal(err)
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
		server := NewFileServer(t, val.Path)
		ctl, err := rain.New(server.URL, rain.WithRoutineCount(3), rain.WithRoutineSize(1024<<10)).Run()
		if err != nil {
			t.Fatal(err)
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
		server := NewFileServer(t, val.Path)
		ctl, err := rain.New(server.URL, rain.WithSpeedLimit(1024<<10), rain.WithBar()).Run()
		if err != nil {
			t.Fatal(err)
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
		server := NewFileServer(t, val.Path)
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
		server := NewFileServer(t, val.Path)

		// 创建同名文件
		os.Mkdir("./tmp", os.ModePerm)
		f, err := os.Create("./tmp/" + val.Name)
		if err != nil {
			t.Fatal(key, err)
		}
		f.Close()

		ctl, err := rain.New(server.URL, rain.WithAutoFileRenaming(true)).Run()
		if err != nil {
			t.Fatal(err)
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

// TestControlReuse 测试复用 control
func TestControlReuse(t *testing.T) {
	Init()
	for key, val := range tf {
		server := NewFileServer(t, val.Path)
		ctl, err := rain.New(server.URL, rain.WithSpeedLimit(1024<<10), rain.WithBar()).Start()
		if err != nil {
			t.Fatal(key, err)
		}
		go func() {
			time.Sleep(time.Second)
			ctl.Close()
		}()
		err = ctl.Wait()
		if err != nil {
			t.Fatal(key, err)
		}
		_, err = ctl.Run()
		if err != nil {
			t.Fatal(key, err)
		}
		// 验证文件完整性
		if FileMD5(ctl.Outpath()) != val.MD5 {
			t.Fatal(key, "md5 错误")
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
		server := NewFileServer(t, val.Path)
		u, err := url.Parse("http://" + proxy)
		if err != nil {
			t.Fatal(err)
		}
		rain.SetProxy(http.ProxyURL(u))
		ctl, err := rain.New(server.URL).Run()
		if err != nil {
			t.Fatal(key, err)
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

// TestSpeed 测试限速
func TestSpeed(t *testing.T) {
	Init()
	for key, val := range tf {
		server := NewFileServer(t, val.Path)
		ctl := rain.New(server.URL, rain.WithSpeedLimit(1))
		ctl.Start()
		// 定时取消下载
		go func() {
			time.Sleep(time.Second * 2)
			ctl.Close()
		}()
		ctl.Wait()
		// 是否为限速下载
		stat, _ := os.Stat(ctl.Outpath())
		if stat.Size() > rain.COPY_BUFFER_SIZE*5 || stat.Size() < rain.COPY_BUFFER_SIZE {
			t.Fatal("限速下载失败")
		}
		ctl.SetSpeedLimit(1024 << 10)
		// 重新开始下载
		_, err := ctl.Start()
		if err != nil {
			t.Fatal(err)
		}
		// 测试中途修改下载速度
		go func() {
			time.Sleep(time.Second * 2)
			// ctl.SetSpeedLimit(1024 << 10)
		}()
		ctl.Wait()
		if ctl.Error() != nil {
			t.Fatal(ctl.Error())
		}
		// 验证文件完整性
		if FileMD5(ctl.Outpath()) != val.MD5 {
			t.Fatal(key, "md5 错误")
		}
	}
}

type EventExtend struct {
	ChangeCount int
	ErrorCount  int
	CloseCount  int
	FinishCount int
}

func (ee *EventExtend) Change(stat *rain.EventExtend) {
	ee.ChangeCount++
	fmt.Println("change", stat.Progress)
}

func (ee *EventExtend) Error(stat *rain.EventExtend) {
	ee.ErrorCount++
	fmt.Println("error", stat.Error)
}

func (ee *EventExtend) Close(stat *rain.EventExtend) {
	ee.CloseCount++
	fmt.Println("close")
}

func (ee *EventExtend) Finish(stat *rain.EventExtend) {
	ee.FinishCount++
	fmt.Println("finish", stat.Progress)
}

var _ rain.ProgressEventExtend = &EventExtend{}

// TestEventExtend 测试扩展事件
func TestEventExtend(t *testing.T) {
	Init()
	for key, val := range tf {
		server := NewFileServer(t, val.Path)
		testEE := &EventExtend{}

		ctl, err := rain.New(
			server.URL,
			rain.WithSpeedLimit(1024<<10),
			rain.WithTimeout(time.Second),
			rain.WithEventExtend(testEE),
		).Start()
		if err != nil {
			t.Fatal(err)
		}

		// 中途暂停
		go func() {
			time.Sleep(time.Millisecond * 500)
			ctl.Close()
		}()
		// 等待下载完成
		err = ctl.Wait()
		if err != nil {
			t.Fatal(err)
		}
		// 重新开始下载
		_, err = ctl.Run()
		if err != nil && !errors.Is(err, context.DeadlineExceeded) {
			t.Fatal(err)
		}
		// 重新开始下载
		ctl, err = rain.New(
			server.URL,
			rain.WithSpeedLimit(1024<<10),
			rain.WithEventExtend(testEE),
		).Run()
		if err != nil {
			t.Fatal(err)
		}
		// 验证文件完整性
		if FileMD5(ctl.Outpath()) != val.MD5 {
			t.Fatal(key, "md5 错误")
		}
		// 验证扩展数量
		if testEE.ChangeCount == 0 {
			t.Fatal(key, "testEE.ChangeCount < 1")
		}
		if testEE.ErrorCount != 1 {
			t.Fatal(key, "testEE.ErrorCount != 1")
		}
		if testEE.CloseCount != 1 {
			t.Fatal(key, "testEE.CloseCount != 1")
		}
		if testEE.FinishCount != 1 {
			t.Fatal(key, "testEE.FinishCount != 1")
		}
	}
}

// 测试磁盘缓冲区
func TestDiskche(t *testing.T) {
	Init()
	for key, val := range tf {
		server := NewFileServer(t, val.Path)
		ctl, err := rain.New(server.URL, rain.WithDiskCache(-1)).Run()
		if err != nil {
			t.Fatal(err)
		}
		if FileMD5(ctl.Outpath()) != val.MD5 {
			t.Fatal(key, "md5 错误")
		}
	}
}

// 测试请求参数
func TestRequest(t *testing.T) {
	Init()
	for key, val := range tf {
		var (
			testbody, testhead, testmethod = true, true, true
		)
		server := NewFileServer(t, val.Path, func(w http.ResponseWriter, r *http.Request) {
			data, _ := io.ReadAll(r.Body)
			if string(data) != "test body" {
				testbody = false
			}
			if r.Header.Get("test") != "header" {
				testhead = false
			}
			if r.Method != http.MethodPost {
				testmethod = false
			}
		})
		ctl, err := rain.New(
			server.URL,
			rain.WithBody(bytes.NewBufferString("test body")),
			rain.WithHeader(func(h http.Header) {
				h.Add("test", "header")
			}),
			rain.WithMethod(http.MethodPost),
		).Run()
		if err != nil {
			t.Fatal(err)
		}
		if FileMD5(ctl.Outpath()) != val.MD5 {
			t.Fatal(key, "md5 错误")
		}
		if !testbody {
			t.Fatal(key, "test body")
		}
		if !testhead {
			t.Fatal(key, "test testhead")
		}
		if !testmethod {
			t.Fatal(key, "test method")
		}
	}
}

// 测试输出文件
func TestOutfile(t *testing.T) {
	Init()
	for key, val := range tf {
		server := NewFileServer(t, val.Path)
		ctl, err := rain.New(
			server.URL,
			rain.WithCreateDir(false),
			rain.WithPerm(0700),
			rain.WithOutname("outname.mp4"),
		).Run()
		if err != nil && !os.IsNotExist(err) {
			t.Fatal(err)
		}
		if err == nil {
			t.Fatal("成功创建了文件夹")
		}
		os.Mkdir("./tmp", os.ModePerm)
		_, err = ctl.Run()
		if err != nil {
			t.Fatal(err)
		}
		if FileMD5(ctl.Outpath()) != val.MD5 {
			t.Fatal(key, "md5 错误")
		}
		stat, _ := os.Stat(ctl.Outpath())
		if stat.Mode().Perm() != 0700 {
			t.Fatal(key, "文件权限设置失败")
		}
		if stat.Name() != "outname.mp4" {
			t.Fatal(key, "文件名设置失败")
		}
	}
}
