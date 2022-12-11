## rain

rain 实现了方便下载 HTTP 协议文件的特性，比如：多协程、断点续传、自动重命名、限速等。
## 安装

使用 go get 安装 rain

``` sh
go get -u github.com/rock-rabbit/rain
```
 
## 特性

- 多协程
- 断点续传
- 限速下载
- 文件重命名
- 非法字符过滤
- 磁盘缓冲区
- 下载进度监听



## 使用方法

下载到执行目录，并且显示命令行进度条

``` golang
package main

import "github.com/rock-rabbit/rain"

func main() {
	ctl := rain.New(
        "https://sample-videos.com/video123/mp4/720/big_buck_bunny_720p_1mb.mp4",
        rain.WithEvent(rain.NewBar()),
    )
	err := <-ctl.Run()
	if err != nil {
		panic(err)
	}
	fmt.Printf("下载完成：%s\n", ctl.Outpath())
}
```


## 心得

这个项目重构了许多次，每次重构都有新的收获，加深了对于程序设计的理解。

项目第一版构建完成时，程序没有很多的结构，都是一个一个方法串联起来实现的功能，这让之后的功能新增和修改有很大的负担。

现在项目每个功能都有单独的结构或者方法去负责，也就是说实现了高内聚，这方便了对项目的管理。