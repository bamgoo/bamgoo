package bamgoo

import (
	"fmt"
	"regexp"
	"strings"
	"sync"

	. "github.com/bamgoo/base"
)

var (
	basic = &basicModule{
		languages: make(map[string]Language, 0),
		strings:   make(Strings, 0),

		states:   make(States, 0),
		mimes:    make(Mimes, 0),
		regulars: make(Regulars, 0),
		types:    make(map[string]Type, 0),
	}
)

type (
	// basicModule 是基础模块
	// 主要用功能是 状态、多语言字串、MIME类型、正则表达式等等
	basicModule struct {
		mutex     sync.Mutex
		languages map[string]Language
		strings   Strings

		//存储所有状态定义
		states States
		// mimes MIME集合
		mimes Mimes
		// regulars 正则表达式集合
		regulars Regulars
		// types 参数类型集合
		types map[string]Type
	}

	// 注意，以下几个类型，不能使用 xxx = map[xxx]yy 的方法定义
	// 因为无法使用.(type)来断言类型

	// State 状态定义，方便注册
	State  int
	States map[string]State

	// MIME mimetype集合
	Mime  []string
	Mimes map[string]Mime

	// Regular 正则表达式，方便注册
	Regular  []string
	Regulars map[string]Regular

	//多语言配置
	Language struct {
		// Name 语言名称
		Name string
		// Desc 语言说明
		Desc string
		// Accepts 匹配的语言
		// 比如，znCN, zh, zh-CN 等自动匹配
		Accepts []string
		// Strings 当前语言是字符串列表
		Strings Strings
	}
	Strings map[string]string

	// Type 类型定义
	Type struct {
		// Name 类型名称
		Name string

		// Desc 类型说明
		Desc string

		// Alias 类型别名
		Alias []string

		// Check 类型检查方法
		Check TypeCheckFunc

		// Convert 类型转换方法
		Convert TypeConvertFunc
	}

	TypeCheckFunc   func(Any, Var) bool
	TypeConvertFunc func(Any, Var) Any
)

func (this *basicModule) Register(name string, value Any) {
	switch val := value.(type) {
	case Language:
		this.RegisterLanguage(name, val)
	case Strings:
		this.RegisterStrings(name, val)
	case State:
		this.RegisterState(name, val)
	case States:
		this.RegisterStates(val)
	case Mime:
		this.RegisterMime(name, val)
	case Mimes:
		this.RegisterMimes(val)
	case Regular:
		this.RegisterRegular(name, val)
	case Regulars:
		this.RegisterRegulars(val)
	case Type:
		this.RegisterType(name, val)
	}
}

// RegisterLanguage 注册语言
func (this *basicModule) RegisterLanguage(name string, config Language) {
	if config.Strings == nil {
		config.Strings = make(Strings, 0)
	}

	if bamgoo.Override() {
		this.languages[name] = config
	} else {
		if _, ok := this.languages[name]; ok == false {
			this.languages[name] = config
		}
	}
}

// RegisterStrings 注册语言字串
func (this *basicModule) RegisterStrings(name string, config Strings) {
	// 对于不存在的语言，先自动来一个
	if _, ok := this.languages[name]; ok == false {
		this.languages[name] = Language{
			Name: name, Desc: name, Accepts: []string{},
			Strings: make(Strings, 0),
		}
	}

	if lang, ok := this.languages[name]; ok {
		for key, str := range config {
			key = strings.Replace(key, ".", "_", -1)
			if bamgoo.Override() {
				lang.Strings[key] = str
			} else {
				if _, ok := lang.Strings[key]; ok == false {
					lang.Strings[key] = str
				}
			}
		}
	}
}

// RegisterState 注册状态
func (this *basicModule) RegisterState(name string, config State) {
	if bamgoo.Override() {
		this.states[name] = config
	} else {
		if _, ok := this.states[name]; ok == false {
			this.states[name] = config
		}
	}
}

// RegisterStates 批量注册状态
func (this *basicModule) RegisterStates(config States) {
	for key, val := range config {
		this.RegisterState(key, State(val))
	}
}

// RegisterMime 注册Mimetype
func (this *basicModule) RegisterMime(name string, config Mime) {
	this.mutex.Lock()
	defer this.mutex.Unlock()

	if bamgoo.Override() {
		this.mimes[name] = config
	} else {
		if _, ok := this.mimes[name]; ok == false {
			this.mimes[name] = config
		}
	}
}

// RegisterMimes 批量注册Mimetype
func (this *basicModule) RegisterMimes(config Mimes) {
	for key, val := range config {
		this.RegisterMime(key, val)
	}
}

// RegisterRegular 注册正则表达式
func (this *basicModule) RegisterRegular(name string, config Regular) {
	this.mutex.Lock()
	defer this.mutex.Unlock()

	if bamgoo.Override() {
		this.regulars[name] = config
	} else {
		if _, ok := this.regulars[name]; ok == false {
			this.regulars[name] = config
		}
	}
}

// RegisterRegulars 批量注册正则表达式
func (this *basicModule) RegisterRegulars(config Regulars) {
	for key, val := range config {
		this.RegisterRegular(key, val)
	}
}

// RegisterType 注册类型
func (this *basicModule) RegisterType(name string, config Type) {
	this.mutex.Lock()
	defer this.mutex.Unlock()

	alias := make([]string, 0)
	if name != "" {
		alias = append(alias, name)
	}
	if config.Alias != nil {
		alias = append(alias, config.Alias...)
	}

	for _, key := range alias {
		if bamgoo.Override() {
			this.types[key] = config
		} else {
			if _, ok := this.types[key]; ok == false {
				this.types[key] = config
			}
		}
	}
}

func (this *basicModule) Config(Map) {}
func (this *basicModule) Setup()     {}
func (this *basicModule) Open()      {}
func (this *basicModule) Start()     {}
func (this *basicModule) Stop()      {}
func (this *basicModule) Close()     {}

// StateCode 获取状态的代码
// defs 可指定默认code，不存在时将返回默认code
func (this *basicModule) StateCode(state string, defs ...int) int {
	if code, ok := this.states[state]; ok {
		return int(code)
	}
	if len(defs) > 0 {
		return defs[0]
	}
	return -1
}

// 结果列表
func (this *basicModule) Results(langs ...string) map[State]string {
	lang := DEFAULT
	if len(langs) > 0 {
		lang = langs[0]
	}

	codes := map[State]string{}
	for key, state := range this.states {
		codes[state] = this.String(lang, key)
	}
	return codes
}

func (this *basicModule) Languages() map[string]Language {
	langs := make(map[string]Language, 0)
	for key, val := range this.languages {
		langs[key] = val
	}
	return langs
}

func (this *basicModule) String(lang, key string, args ...Any) string {
	if lang == "" {
		lang = DEFAULT
	}

	//把所有语言字串的.都替换成_
	key = strings.Replace(key, ".", "_", -1)

	langStr := key
	if cfg, ok := this.languages[lang]; ok {
		if str, ok := cfg.Strings[key]; ok {
			langStr = str
		}
	} else if cfg, ok := this.languages[DEFAULT]; ok {
		if str, ok := cfg.Strings[key]; ok {
			langStr = str
		}
	} else {
		langStr = key
	}

	if len(args) > 0 {
		ccc := strings.Count(langStr, "%") - strings.Count(langStr, "%%")
		if ccc == len(args) {
			return fmt.Sprintf(langStr, args...)
		}
	}
	return langStr
}

// Extension 按MIME取扩展名
// defs 为默认值，如果找不到对英的mime，则返回默认
func (this *basicModule) Extension(mime string, defs ...string) string {
	for ext, mmms := range this.mimes {
		for _, mmm := range mmms {
			if strings.ToLower(mmm) == strings.ToLower(mime) {
				return ext
			}
		}
	}
	if len(defs) > 0 {
		return defs[0]
	}
	return ""
}

// Mimetype 按扩展名拿 MIMEType
// defs 为默认值，如果找不到对应的mime，则返回默认
func (this *basicModule) Mimetype(ext string, defs ...string) string {
	if strings.Contains(ext, "/") {
		return ext
	}

	//去掉点.
	if strings.HasPrefix(ext, ".") {
		ext = strings.TrimPrefix(ext, ".")
	}

	if mimes, ok := this.mimes[ext]; ok && len(mimes) > 0 {
		return mimes[0]
	}
	// 如果定义了*，所有不匹配的扩展名，都返回*
	if mimes, ok := this.mimes["*"]; ok && len(mimes) > 0 {
		return mimes[0]
	}
	if len(defs) > 0 {
		return defs[0]
	}

	return "application/octet-stream"
}

// Expressions 获取正则的表达式
func (this *basicModule) Expressions(name string, defs ...string) []string {
	if exps, ok := this.regulars[name]; ok {
		return exps
	}
	return defs
}

// Match 正则匹配
func (this *basicModule) Match(regular, value string) bool {
	matchs := this.Expressions(regular)
	for _, v := range matchs {
		regx := regexp.MustCompile(v)
		if regx.MatchString(value) {
			return true
		}
	}
	return false
}

// Types 获取所有类型
func (this *basicModule) Types() map[string]Type {
	types := map[string]Type{}
	for k, v := range this.types {
		types[k] = v
	}
	return types
}

// typeDefaultCheck 默认的类型校验方法
// 直接转到正则去匹配
func (this *basicModule) typeDefaultCheck(value Any, config Var) bool {
	return this.Match(config.Type, fmt.Sprintf("%s", value))
}

// typeDefaultConvert 默认值包装方法
func (this *basicModule) typeDefaultConvert(value Any, config Var) Any {
	return fmt.Sprintf("%s", value)
}

// typeCheck 获取类型的校验方法
func (this *basicModule) typeCheck(name string) TypeCheckFunc {
	if config, ok := this.types[name]; ok {
		if config.Check != nil {
			return config.Check
		}
	}
	return this.typeDefaultCheck
}

// typeConvert 获取类型的值包装方法
func (this *basicModule) typeConvert(name string) TypeConvertFunc {
	if config, ok := this.types[name]; ok {
		if config.Convert != nil {
			return config.Convert
		}
	}
	return this.typeDefaultConvert
}

// typeConvert 获取类型的校验和值包装方法
func (this *basicModule) typeMethod(name string) (TypeCheckFunc, TypeConvertFunc) {
	return this.typeCheck(name), this.typeConvert(name)
}

// StateCode 返回状态码
func StateCode(name string, defs ...int) int {
	return basic.StateCode(name, defs...)
}
func Results(langs ...string) map[State]string {
	return basic.Results(langs...)
}

// Mimetype 按扩展名获取 MIME 中的 类型
func Mimetype(ext string, defs ...string) string {
	return basic.Mimetype(ext, defs...)
}

// Extension 按MIMEType获取扩展名
func Extension(mime string, defs ...string) string {
	return basic.Extension(mime, defs...)
}

func Languages() map[string]Language {
	return basic.Languages()
}

// func Strings(lang string) Strs {
// 	return basic.Strings(lang)
// }

// String 获取多语言字串
func String(lang, name string, args ...Any) string {
	return basic.String(lang, name, args...)
}

// Expressions 获取正则的表达式
func Expressions(name string, defs ...string) []string {
	return basic.Expressions(name, defs...)
}

// Match 正则做匹配校验
func Match(regular, value string) bool {
	return basic.Match(regular, value)
}

// Types 获取所有类型定义
func Types() map[string]Type {
	return basic.Types()
}
