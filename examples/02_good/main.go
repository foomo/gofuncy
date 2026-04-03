package main

import (
	"context"
	"fmt"
	"time"

	"github.com/foomo/gofuncy"
)

func main() {
	msg := make(chan string, 10)

	fmt.Println("start")

	ctx := gofuncy.Ctx(context.Background()).Root()

	gofuncy.Go(ctx, send(msg), gofuncy.GoOption().WithName("sender-a").WithTracing())
	gofuncy.Go(ctx, send(msg), gofuncy.GoOption().WithName("sender-b"))

	gofuncy.Go(ctx, receive(msg), gofuncy.GoOption().WithName("receiver-c"))

	time.Sleep(3 * time.Second)
}

func send(msg chan string) gofuncy.Func {
	return func(ctx context.Context) error {
		for {
			msg <- fmt.Sprintf("Hello World #%s", gofuncy.Ctx(ctx).Name())
			time.Sleep(300 * time.Millisecond)
		}
	}
}

func receive(msg chan string) gofuncy.Func {
	return func(ctx context.Context) error {
		for m := range msg {
			fmt.Println(m, "by", gofuncy.Ctx(ctx).Name())
			// fmt.Println(m, len(msg))
			time.Sleep(time.Second)
		}
		return nil
	}
}
