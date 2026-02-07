package main

import (
	"fmt"

	. "github.com/bamgoo/bamgoo/base"
	_ "github.com/bamgoo/bamgoo/builtin"

	"github.com/bamgoo/bamgoo"
)

func main() {
	bamgoo.Go()
}

func init() {
	bamgoo.Register(bamgoo.START, bamgoo.Trigger{
		Name: "启动", Desc: "启动",
		Action: func(ctx *bamgoo.Context) {
			data := ctx.Invoke("test.get", Map{"msg": "msg from examples."})
			res := ctx.Result()

			fmt.Println("ssss", res, data)
		},
	})

}
