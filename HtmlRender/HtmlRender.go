package HtmlRender

import (
	"bufio"
	"fmt"
	"log"
	"os"
	"reflect"
	"regexp"
	"strings"
)

// region 错误定义相关

type HtmlParseError struct {
	msg  string
	line string
}

func (e *HtmlParseError) Error() string {
	return fmt.Sprintf("html模板解析错误 -> %s; line: %s", e.msg, e.line)
}

// endregion

// region 模板渲染相关

const scriptHeader = `package main

import (
	"strings"
)

func main(){
var builder strings.Builder
defer builder.Reset()
`

const scriptFooter = `
println(builder.String())
}
`

type HtmlRender struct {
	templatePath  string
	scriptPath    string
	scriptBuilder strings.Builder
}

// RenderHtml 解析文件并生成go脚本. htmlPath为html模板文件路径, context为html模板的数据结构体
func (render HtmlRender) RenderHtml(htmlPath string, context interface{}) error {
	if !render.isExists(htmlPath) {
		return os.ErrNotExist
	}

	//var builder strings.Builder
	//defer builder.Reset()

	// 脚本开始
	//builder.WriteString(scriptHeader)
	// 创建脚本的 internal object
	// TODO 如何处理要渲染的数据
	// 模板转go代码
	err := render.ParseHtmlFile(htmlPath)
	if err != nil {
		return err
	}
	// 脚本结束
	//builder.WriteString(scriptFooter)

	// 写入文件
	//writer, err := os.Create("./a.go")
	//if err != nil {
	//	return err
	//}
	//_, err = writer.WriteString(builder.String())
	//if err != nil {
	//	return err
	//}
	return nil
}

// 文件是否存在
func (render HtmlRender) isExists(_path string) bool {
	_, err := os.Stat(_path)
	return err == nil
}

const exprRegex = "{{[\\s|\\S|\\w|\\p{Han}]+}}"
const writeString = "builder.WriteString(`%s`)\n"
const writeVariableStart = "builder.WriteString(fmt.Sprintf(\"%s\""
const writeVariableEnd = ", %s))\n"
const variableDefine = "%s\n"
const exprStart = "%s {\n"
const exprEnd = "}\n"
const exprElse = "} %s {\n"

// ParseHtmlFile 解析HTML文件 htmlPath html模板路径
func (render HtmlRender) ParseHtmlFile(htmlPath string) error {
	file, err := os.Open(htmlPath)
	if err != nil {
		return err
	}
	defer func(file *os.File) {
		err := file.Close()
		if err != nil {
			log.Fatal(err)
		}
	}(file)
	re, err := regexp.Compile(exprRegex)
	if err != nil {
		return err
	}
	// html 转 脚本
	reader := bufio.NewScanner(file)
	for reader.Scan() {
		line := reader.Text()
		/*
		 * 通过正则找到表达式字符串
		 * 切分表达式字符串
		 * 将非表达式部分直接写入 builder
		 */
		match := re.FindStringIndex(line)
		if len(match) == 0 {
			// 普通的字符串

			render.scriptBuilder.WriteString(fmt.Sprintf(writeString, line))
		} else if len(match) == 2 {
			_start := line[0:match[0]]       // 开头
			_expr := line[match[0]:match[1]] // 代码
			_end := line[match[1]:]          // 结尾
			if len(_start) > 0 {
				render.scriptBuilder.WriteString(fmt.Sprintf(writeString, _start))
			}
			if strings.HasPrefix(_expr, "{{%") && strings.HasSuffix(_expr, "%}}") {
				// 控制语句
				_realExpr := _expr[3 : len(_expr)-3]
				_realExpr = strings.TrimSpace(_realExpr)
				if _realExpr == "end" {
					render.scriptBuilder.WriteString(exprEnd)
				} else if strings.HasPrefix(_realExpr, "else") {
					render.scriptBuilder.WriteString(fmt.Sprintf(exprElse, _realExpr))
				} else {
					render.scriptBuilder.WriteString(fmt.Sprintf(exprStart, _realExpr))
				}
			} else if strings.HasPrefix(_expr, "{{#") && strings.HasSuffix(_expr, "#}}") {
				// 变量定义
				_realExpr := _expr[3 : len(_expr)-3]
				render.scriptBuilder.WriteString(fmt.Sprintf(variableDefine, _realExpr))

			} else if strings.HasPrefix(_expr, "{{") && strings.HasSuffix(_expr, "}}") {
				// 单纯访问变量
				_varName := _expr[2 : len(_expr)-2]
				render.scriptBuilder.WriteString(writeVariableStart)
				render.scriptBuilder.WriteString(fmt.Sprintf(writeVariableEnd, _varName))
			} else {
				return &HtmlParseError{msg: "表达式格式错误", line: line}
			}
			if len(_end) > 0 {
				render.scriptBuilder.WriteString(fmt.Sprintf(writeString, _end))
			}
		} else {
			// 正则匹配出问题了??
			return &HtmlParseError{msg: "正则匹配错误", line: line}
		}
	}
	if err := reader.Err(); err != nil {
		return err
	}
	return nil
}

func (render HtmlRender) dumpDataStruct(typeOfT reflect.Type) []string {
	structDefineList := make([]string, 0)
	var builder strings.Builder
	builder.WriteString(fmt.Sprintf("type %s struct {\n", typeOfT.Name()))
	for i := 0; i < typeOfT.NumField(); i++ {
		field := typeOfT.Field(i)
		fieldName := field.Name
		fieldKind := field.Type.Kind()
		switch fieldKind {
		case reflect.Struct:
			childStruct := render.dumpDataStruct(field.Type)
			structDefineList = append(structDefineList, childStruct...)
			builder.WriteString(fmt.Sprintf("\t%s %s\n", fieldName, field.Type.Name()))
			break
		case reflect.Map:
			mapKey := field.Type.Key()
			mapEl := field.Type.Elem()
			if mapKey.Kind() == reflect.Struct {
				childStruct := render.dumpDataStruct(mapKey)
				structDefineList = append(structDefineList, childStruct...)
			}
			if mapEl.Kind() == reflect.Struct {
				childStruct := render.dumpDataStruct(mapEl)
				structDefineList = append(structDefineList, childStruct...)
			}
			builder.WriteString(fmt.Sprintf("\t%s map[%s]%s\n", fieldName, mapKey.Name(), mapEl.Name()))
			break
		case reflect.Slice:
			el := field.Type.Elem()
			if el.Kind() == reflect.Struct {
				elStruct := render.dumpDataStruct(el)
				structDefineList = append(structDefineList, elStruct...)
			}
			builder.WriteString(fmt.Sprintf("\t%s []%s\n", fieldName, el.Name()))
			break
		case reflect.Array:
			arrLen := field.Type.Len()
			el := field.Type.Elem()
			if el.Kind() == reflect.Struct {
				elStruct := render.dumpDataStruct(el)
				structDefineList = append(structDefineList, elStruct...)
			}
			builder.WriteString(fmt.Sprintf("\t%s [%d]%s\n", fieldName, arrLen, el.Name()))
			break
		default:
			builder.WriteString(fmt.Sprintf("\t%s %s\n", fieldName, fieldKind))
		}
	}
	builder.WriteString("}")
	structDefineList = append(structDefineList, builder.String())
	return structDefineList
}

// endregion
