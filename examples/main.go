package main

import (
	"fmt"
	"time"

	"github.com/bamgoo/bamgoo"
)

func main() {
	bamgoo.Go()
}

func init() {

	bamgoo.Register("nats", bamgoo.BusConfig{
		Driver: "nats",
	})

	bamgoo.Register(bamgoo.START, bamgoo.Trigger{
		Name: "启动", Desc: "启动",
		Action: func(ctx *bamgoo.Context) {
			start := time.Now()

			for range 100000 {
				data := ctx.Invoke("test.get")
				fmt.Println("start....", data)
			}
			end := time.Now()
			fmt.Println("cost:", end.Sub(start))
		},
	})

}
