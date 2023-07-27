# SimpleHtmlRender

## 简介
golang 实现的简易html模板渲染  
模板中可以嵌入 golang 代码

## 模板语法

详见 [template.html](template.html)  
简述:

```
{{OBJECT.variable}}  // 使用变量
{{#var myMap map[int32]string#}} // 定义变量
{{%if condition%}}  // if语句
{{%else%}}  // else语句
{{%end%}}  // 结束语句
{{%for condition%}}  // for语句
```

## 用法

```go
package main

import (
	"HtmlRender/HtmlRender"
	"fmt"
	"os"
)

type School struct {
	Name string
	Addr string
}

type UserInfo struct {
	id     int8
	Name   string
	Email  string
	Info   map[string]string
	Fake   []string
	School map[string][]School
	arrMap map[[2]int32]string
}

func main() {
	render := HtmlRender.HtmlRender{}
	err := render.ParseHtmlFile("./template.html")
	if err != nil {
		fmt.Println(err.Error())
	}
	user := UserInfo{
		id:    1,
		Name:  "test",
		Email: "temp@temp.com",
		Info: map[string]string{
			"age":  "18",
			"city": "shanghai",
			"job":  "coder",
		},
		Fake: []string{"1", "2", "3"},
		School: map[string][]School{
			"小学": {
				{"小学1", "小学1地址"},
				{"小学2", "小学2地址"},
			},
			"中学": {
				{"中学1", "中学1地址"},
			},
		},
	}
	var output string
	output, err = render.RenderHtml(user)
	if err != nil {
		fmt.Println(err.Error())
		return
	}
	file, err := os.Create("./output.html")
	if err != nil {
		fmt.Println(err.Error())
		return
	}
	_, err = file.WriteString(output)
	if err != nil {
		fmt.Println(err.Error())
		return
	}
}

```
