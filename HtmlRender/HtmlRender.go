package HtmlRender

import (
	"bufio"
	"encoding/json"
	"fmt"
	"log"
	"os"
	"os/exec"
	"reflect"
	"regexp"
	"strconv"
	"strings"
	"time"
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

// region
type void struct{}
type Set struct {
	version    uint64
	preVersion uint64
	values     map[string]void
	cache      []string
	freeze     bool
}

func (s *Set) Init() {
	s.values = make(map[string]void)
	s.freeze = false
}

func (s *Set) Add(val string) {
	if s.values == nil {
		s.values = make(map[string]void)
	}
	if _, ok := s.values[val]; ok {
		return
	}
	var empty void
	s.values[val] = empty
	if !s.freeze {
		s.version += 1
	}
}

func (s *Set) AddAll(values []string) {
	s.freeze = true
	for _, v := range values {
		s.Add(v)
	}
	s.version += 1
	s.freeze = false
}

func (s *Set) Del(val string) {
	if s.values == nil {
		return
	}
	if _, ok := s.values[val]; !ok {
		return
	}
	delete(s.values, val)
	s.version += 1
}

func (s *Set) ToSlice() []string {
	if s.values == nil {
		return nil
	}
	if s.preVersion == s.version {
		return s.cache
	}
	s.cache = make([]string, 0)
	for k := range s.values {
		s.cache = append(s.cache, k)
	}
	s.preVersion = s.version
	return s.cache
}

// endregion

// region 模板渲染相关

const scriptHeader = `package main

import (
	"encoding/json"
	"fmt"
	"strings"
	%s
)
`

const mainFuncStart = `func main(){
var builder strings.Builder
defer builder.Reset()
`

const mainFuncEnd = `
println(builder.String())
}
`

type HtmlRender struct {
	templatePath   string
	scriptBuilder  strings.Builder
	importPackages *Set
}

const exprRegex = "{.*?}"
const writeString = "builder.WriteString(`%s`)\n"
const writeVariableStart = "builder.WriteString(fmt.Sprintf(\"%s\""
const writeVariableEnd = ", %s))\n"

// ParseHtmlFile 解析html模板文件 htmlPath为html模板文件路径
func (render *HtmlRender) ParseHtmlFile(htmlPath string) error {
	if isExists, err := render.isExists(htmlPath); !isExists {
		return &HtmlParseError{msg: err.Error(), line: ""}
	}
	render.templatePath = htmlPath
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
	// 初始化相关变量
	render.scriptBuilder.Reset()
	render.importPackages = &Set{}
	// html 转 脚本
	reader := bufio.NewScanner(file)
	findGolangCode := false
	resetCodeFlag := false
	for reader.Scan() {
		line := reader.Text()
		line = strings.TrimSpace(line)
		if line == "" {
			continue
		}
		/*
		 * 通过正则找到表达式字符串
		 * 切分表达式字符串
		 * 将非表达式部分直接写入 builder
		 */
		matches := re.FindAllStringIndex(line, -1)
		if len(matches) == 0 {
			// 普通的字符串
			render.scriptBuilder.WriteString(fmt.Sprintf(writeString, line))
		} else if len(matches) > 0 {
			htmlCodeList := make([]string, 0)
			htmlCodeList = append(htmlCodeList, strings.TrimSpace(line[:matches[0][0]]))
			prevEnd := -1
			for _, tuple := range matches {
				start := tuple[0]
				end := tuple[1]
				if prevEnd != -1 {
					htmlCodeList = append(htmlCodeList, strings.TrimSpace(line[prevEnd:start]))
				}
				prevEnd = end
				htmlCodeList = append(htmlCodeList, strings.TrimSpace(line[start:end]))
			}
			htmlCodeList = append(htmlCodeList, strings.TrimSpace(line[matches[len(matches)-1][1]:]))

			for _, code := range htmlCodeList {
				if !findGolangCode && strings.HasPrefix(code, "{%") {
					code = code[2:]
					findGolangCode = true
				}
				if findGolangCode && strings.HasSuffix(code, "%}") {
					code = code[:len(code)-2]
					resetCodeFlag = true
				}
				if findGolangCode {
					// 纯 golang 代码
					if strings.HasPrefix(code, "if") || strings.HasPrefix(code, "for") {
						render.scriptBuilder.WriteString(fmt.Sprintf("%s {", code))
					} else if strings.HasPrefix(code, "else") || strings.HasPrefix(code, "else if") {
						render.scriptBuilder.WriteString(fmt.Sprintf("} %s {", code))
					} else if code == "end" {
						render.scriptBuilder.WriteString("}")
					} else {
						render.scriptBuilder.WriteString(code)
					}
					render.scriptBuilder.WriteString(" \n")
				} else if strings.HasPrefix(code, "{i") && strings.HasSuffix(code, "}") {
					// 包体导入
					importContent := code[2 : len(code)-1]
					imports := strings.SplitN(importContent, ",", -1)
					render.importPackages.AddAll(imports)
				} else if strings.HasPrefix(code, "{=") && strings.HasSuffix(code, "}") {
					// 访问变量
					_varName := code[2 : len(code)-1]
					render.scriptBuilder.WriteString(writeVariableStart)
					render.scriptBuilder.WriteString(fmt.Sprintf(writeVariableEnd, _varName))
				} else {
					render.scriptBuilder.WriteString(fmt.Sprintf(writeString, code))
				}

				if resetCodeFlag {
					findGolangCode = false
				}
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

// RenderHtml 解析文件并生成go脚本. context为html模板的数据结构体
func (render *HtmlRender) RenderHtml(context interface{}) (string, error) {
	var builder strings.Builder
	defer builder.Reset()

	// 脚本开始
	var packagesBuilder strings.Builder
	for _, pack := range render.importPackages.ToSlice() {
		packagesBuilder.WriteString(fmt.Sprintf("\"%s\"\n", strings.TrimSpace(pack)))
	}
	builder.WriteString(fmt.Sprintf(scriptHeader, packagesBuilder.String()))
	packagesBuilder.Reset()
	// 创建脚本的 internal object
	contextType := reflect.TypeOf(context)
	structList := render.reflectDataStruct(contextType)
	if len(structList) != 0 {
		builder.WriteString(strings.Join(structList, "\n"))
	}
	builder.WriteString("\n")
	builder.WriteString(mainFuncStart)
	// context json 互转
	contextJson, err := json.Marshal(context)
	if err != nil {
		return "", err
	}
	builder.WriteString(fmt.Sprintf("dataJson := `%s`\n", string(contextJson)))
	builder.WriteString(fmt.Sprintf("OBJECT := %s{}\n", contextType.Name()))
	builder.WriteString("err := json.Unmarshal([]byte(dataJson), &OBJECT)\n")
	builder.WriteString("if err != nil {\n")
	builder.WriteString("\tpanic(err)\n")
	builder.WriteString("}\n")
	// 模板转go代码
	builder.WriteString(render.scriptBuilder.String())
	// 脚本结束
	builder.WriteString(mainFuncEnd)

	// 写入文件
	htmlName := strings.TrimSuffix(render.templatePath, ".html")
	timestamp := strconv.FormatInt(time.Now().UnixMilli(), 10)
	filePath := fmt.Sprintf("./%s_%s.go", htmlName, timestamp)
	// 删除已执行的脚本
	defer func(name string) {
		_ = os.Remove(name)
	}(filePath)
	writer, err := os.Create(filePath)
	if err != nil {
		return "", err
	}
	_, err = writer.WriteString(builder.String())
	if err != nil {
		return "", err
	}
	err = writer.Close()
	if err != nil {
		return "", err
	}
	// go run
	output, err := exec.Command("go", "run", filePath).CombinedOutput()
	return string(output), err
}

// 文件是否存在
func (render *HtmlRender) isExists(_path string) (bool, error) {
	_, err := os.Stat(_path)
	return err == nil, err
}

// reflectMapKV 递归解析map的key/value
func (render *HtmlRender) reflectMapKV(typeOfT reflect.Type) (string, []string) {
	switch typeOfT.Kind() {
	case reflect.Struct:
		childStruct := render.reflectDataStruct(typeOfT)
		return typeOfT.Name(), childStruct
	case reflect.Array:
		arrLen := typeOfT.Len()
		el := typeOfT.Elem()
		if el.Kind() == reflect.Struct {
			elStruct := render.reflectDataStruct(el)
			return fmt.Sprintf("[%d]%s", arrLen, el.Name()), elStruct
		} else {
			return fmt.Sprintf("[%d]%s", arrLen, el.Kind()), nil
		}
	case reflect.Slice:
		el := typeOfT.Elem()
		if el.Kind() == reflect.Struct {
			elStruct := render.reflectDataStruct(el)
			return fmt.Sprintf("[]%s", el.Name()), elStruct
		} else {
			return fmt.Sprintf("[]%s", el.Kind()), nil
		}
	}
	return typeOfT.Kind().String(), nil
}

// 递归解析数据结构体
func (render *HtmlRender) reflectDataStruct(typeOfT reflect.Type) []string {
	structDefineList := make([]string, 0)
	var builder strings.Builder
	builder.WriteString(fmt.Sprintf("type %s struct {\n", typeOfT.Name()))
	for i := 0; i < typeOfT.NumField(); i++ {
		field := typeOfT.Field(i)
		fieldName := field.Name
		fieldKind := field.Type.Kind()
		switch fieldKind {
		case reflect.Struct:
			childStruct := render.reflectDataStruct(field.Type)
			structDefineList = append(structDefineList, childStruct...)
			builder.WriteString(fmt.Sprintf("\t%s %s\n", fieldName, field.Type.Name()))
			break
		case reflect.Map:
			mapKey := field.Type.Key()
			mapEl := field.Type.Elem()
			// map的key/value可能是struct/array/slice: map[2]int32][2]CustomStruct
			keyType, keyStruct := render.reflectMapKV(mapKey)
			if keyStruct != nil && len(keyStruct) > 0 {
				structDefineList = append(structDefineList, keyStruct...)
			}
			elType, elStruct := render.reflectMapKV(mapEl)
			if elStruct != nil && len(elStruct) > 0 {
				structDefineList = append(structDefineList, elStruct...)
			}
			builder.WriteString(fmt.Sprintf("\t%s map[%s]%s\n", fieldName, keyType, elType))
			break
		case reflect.Slice:
			el := field.Type.Elem()
			if el.Kind() == reflect.Struct {
				elStruct := render.reflectDataStruct(el)
				structDefineList = append(structDefineList, elStruct...)
			}
			builder.WriteString(fmt.Sprintf("\t%s []%s\n", fieldName, el.Name()))
			break
		case reflect.Array:
			arrLen := field.Type.Len()
			el := field.Type.Elem()
			if el.Kind() == reflect.Struct {
				elStruct := render.reflectDataStruct(el)
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
