## rain - 超快下载 http 资源

rain 一款 golang 包，可以让你快速实现 http 协议的资源下载，为此 rain 拥有一些方便的特性，比如：多协程、断点续传、自动重命名、限速等。


## 安装

使用 go get 安装 rain

``` sh
go get -u github.com/rock-rabbit/rain
```
 
## 特性

- 多协程分块下载
- 断点下载
- 限速下载
- 文件自动重命名
- 文件名非法字符过滤
- 磁盘缓冲区
- 下载进度和状态监听
- 可自定义的命令行进度条
- 运行时修改配置
- 非阻塞下载

当前测试覆盖： coverage: 78.5% of statements

## 使用方法

**rain 简单的使用方法：**

``` golang
package main

import "github.com/rock-rabbit/rain"

func main() {
	uri :=  "https://sample-videos.com/video123/mp4/720/big_buck_bunny_720p_1mb.mp4"
	ctl, err := rain.New(uri, rain.WithBar()).Run()
	if err != nil {
		panic(err)
	}
	fmt.Printf("下载完成：%s\n", ctl.Outpath())
}


## 项目
以下项目使用到了 rain :

* [rain-service](https://github.com/rock-rabbit/rain-service): rpc 下载服务
* [rain-service-gui](https://github.com/rock-rabbit/rain-service-gui): 基于 rain-service 的跨平台图形界面

