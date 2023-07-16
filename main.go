package main

import (
	"encoding/json"
	"fmt"
	"reflect"
	"strings"
)

type Complex struct {
}

type UserInfo struct {
	id    int8
	Name  string
	Email string
	Info  map[string]string
}

func (user UserInfo) foo() {

}

func ListFields(a interface{}) {
	var builder strings.Builder
	s := reflect.TypeOf(a)
	println(s.Name())
	builder.WriteString(fmt.Sprintf("type %s struct {", s.Name()))
	v := reflect.ValueOf(a)
	for j := 0; j < v.NumField(); j++ {
		f := v.Field(j)
		n := v.Type().Field(j).Name
		tName := f.Type().Name()

		if f.Type().Kind() == reflect.Struct {
			fmt.Printf("type %s struct\n", tName)
		}
		fmt.Printf("Name: %s  Basic Type or Kind: %s  Direct or Custom Type: %s\n", n, f.Kind(), tName)
		builder.WriteString(fmt.Sprintf("\t%s %s\n", n, f.Kind()))
	}
	builder.WriteString("}")
	println(builder.String())
}

func main() {
	//render := HtmlRender.HtmlRender{}
	//err := render.ParseFile("./template.html")
	//if err != nil {
	//	fmt.Println(err.Error())
	//}

	p := UserInfo{Name: "meow", Email: "meow@meow.com", Info: map[string]string{"小学": "XXX", "高中": "asdfaf"}}

	ListFields(p)

	jsonP, err := json.Marshal(p)
	if err != nil {
		println(err.Error())
		return
	}
	var p2 UserInfo
	err = json.Unmarshal(jsonP, &p2)
	if err != nil {
		println(err.Error())
		return
	}
	println(p2.Name)
	//re, _ := regexp.Compile("{{[\\s|\\S|\\w|\\p{Han}]+}}")
	////s := "<h1>你好{{title}}世界</h1>"
	////s := "<h1>{{%if title=\"meowlos\"%}}{{title}}</h1>"
	//s := "<h1>{{%if title=\"喵洛斯\"%}}</h1>"
	////s := "</head>"
	////s := "<h1>{{%if title=\"meowlos\"}</h1>}"
	//match := re.FindStringIndex(s)
	//fmt.Println(match)
	//if len(match) > 1 {
	//	fmt.Println(s[0:match[0]])
	//	fmt.Println(s[match[0]:match[1]])
	//	fmt.Println(s[match[1]:len(s)])
	//
	//	_expr := s[match[0]:match[1]]
	//	_realExpr := _expr[3 : len(_expr)-3]
	//	fmt.Println(_realExpr)
	//}
}
