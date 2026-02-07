package main

import (
	"fmt"

	"github.com/bamgoo/bamgoo"
	. "github.com/bamgoo/bamgoo/base"
	"github.com/bamgoo/bamgoo/bus"
	_ "github.com/bamgoo/bamgoo/bus"
	_ "github.com/bamgoo/bamgoo/bus-default"
	_ "github.com/bamgoo/bamgoo/bus-nats"
	_ "github.com/bamgoo/bamgoo/config"
	_ "github.com/bamgoo/bamgoo/config-file"
	_ "github.com/bamgoo/bamgoo/config-redis"
)

func main() {
	bamgoo.Go()
}

func init() {

	bamgoo.Register("nats", bus.BusConfig{
		Driver: "nats",
	})

	bamgoo.Register(bamgoo.START, bamgoo.Trigger{
		Name: "启动", Desc: "启动",
		Action: func(ctx *bamgoo.Context) {
			data := ctx.Invoke("test.get", Map{"msg": "msg from examples."})
			res := ctx.Result()

			fmt.Println("ssss", res, data)
		},
	})

}
