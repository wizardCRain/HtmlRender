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

详见 [main.go](main.go)
