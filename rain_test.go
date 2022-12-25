package rain_test

import (
	"fmt"
	"testing"
	"time"

	"github.com/rock-rabbit/rain"
)

var task = []struct {
	uri  string
	name string
}{
	{
		uri:  "https://sample-videos.com/video123/mp4/720/big_buck_bunny_720p_1mb.mp4",
		name: "test1m.mp4",
	},
	{
		uri:  "https://sample-videos.com/video123/mp4/720/big_buck_bunny_720p_5mb.mp4",
		name: "test5m.mp4",
	},
	{
		uri:  "https://sample-videos.com/img/Sample-jpg-image-10mb.jpg",
		name: "test10m.jpg",
	},
}

// TestSingleThread 测试单协程下载
func TestSingleThread(t *testing.T) {
	rain.SetRoutineSize(1 << 20)
	for _, v := range task {
		ctl := rain.New(v.uri, rain.WithOutdir("./tmp"), rain.WithOutname(v.name), rain.WithEvent(rain.NewBar()))
		err := <-ctl.Run()
		if err != nil {
			t.Fatal(err)
		}
	}
}

// TestThreads 测试多协程下载
func TestThreads(t *testing.T) {
	rain.SetRoutineSize(1 << 20)
	rain.SetRoutineCount(7)
	for _, v := range task {
		ctl := rain.New(v.uri, rain.WithOutdir("./tmp"), rain.WithOutname(v.name))
		err := <-ctl.Run()
		if err != nil {
			t.Fatal(err)
		}
	}
}

// TestEvent 测试事件
func TestEvent(t *testing.T) {
	rain.SetRoutineSize(1 << 20)
	rain.SetRoutineCount(2)
	for _, v := range task {
		ctl := rain.New(
			v.uri,
			rain.WithOutdir("./tmp"),
			rain.WithOutname(v.name),
			rain.WithEvent(rain.NewBar()),
		)
		err := <-ctl.Run()
		if err != nil {
			t.Fatal(err)
		}
		fmt.Printf("outpath: %s\n", ctl.Outpath())
	}
}

// 测试限速下载
func TestSpeedLimit(t *testing.T) {
	rain.SetSpeedLimit(1024 * 20)
	for _, v := range task {
		ctl := rain.New(
			v.uri,
			rain.WithOutdir("./tmp"),
			rain.WithOutname(v.name),
			rain.WithEvent(rain.NewBar()),
		)
		err := <-ctl.Run()
		if err != nil {
			t.Fail()
		}
		fmt.Printf("outpath: %s\n", ctl.Outpath())
	}
}

// 测试下载中途取消
func TestClose(t *testing.T) {
	v := task[2]

	ctl := rain.New(
		v.uri,
		rain.WithOutdir("./tmp"),
		rain.WithOutname(v.name),
		rain.WithEvent(rain.NewBar()),
	)
	go func() {
		time.Sleep(time.Second * 13)
		ctl.Close()
	}()
	err := <-ctl.Run()
	if err != nil {
		t.Fatal(err)
	}
	fmt.Printf("outpath: %s\n", ctl.Outpath())
}

// TestTimeout 测试超时
func TestTimeout(t *testing.T) {
	// 配置全新的下载器
	downloader := rain.NewRain()
	downloader.SetTimeout(time.Second * 5)
	downloader.SetHeader("referer", "https://www.68wu.cn/")

	// 使用自定义的下载器下载
	ctl := downloader.New(
		"https://sample-videos.com/video123/mp4/720/big_buck_bunny_720p_1mb.mp4",
		rain.WithOutdir("./tmp"),
		rain.WithEvent(rain.NewBar()),
	)
	err := <-ctl.Run()
	if err != nil {
		t.Fatal(err)
	}
	fmt.Printf("下载完成：%s\n", ctl.Outpath())
}
