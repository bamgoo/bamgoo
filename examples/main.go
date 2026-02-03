package main

import (
	"fmt"

	"github.com/bamgoo/bamgoo"
	. "github.com/bamgoo/bamgoo/base"
)

func main() {
	bamgoo.Go()
}

func init() {

	bamgoo.Register("int",bamgoo.Type{
		Name: "测试类型", Desc: "测试类型",
		Check: func(value Any, config Var) bool {
			return true
		},
		Convert: func(value Any, config Var) Any {
			return value
		},
	})

	bamgoo.Register("test.method", bamgoo.Method{
		Name: "测试方法", Desc: "测试方法",
		Action: func(ctx *bamgoo.Context) (Map, Res) {
			return Map{"msg": "hello world"}, nil
		},
	})

	bamgoo.Register("test.service", bamgoo.Service{
		Name: "测试服务", Desc: "测试服务",
		Action: func(ctx *bamgoo.Context) (Map, Res) {
			return Map{"msg": "hello world"}, nil
		},
	})

	bamgoo.Register(bamgoo.START, bamgoo.Trigger{
		Name: "启动", Desc: "启动",
		Action: func(ctx *bamgoo.Context) {
			data := ctx.Invoke("test.method")
			fmt.Println("start....", data)
		},
	})

	bamgoo.Register(bamgoo.START, bamgoo.Trigger{
		Name: "启动2", Desc: "启动2",
		Action: func(ctx *bamgoo.Context) {
			data := ctx.Invoke("test.method")
			fmt.Println("2222 start....", data)
		},
	})

}
