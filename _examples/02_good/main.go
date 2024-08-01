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

	_ = gofuncy.Go(func(ctx context.Context) error {
		return nil
	})
	_ = gofuncy.Go(send(msg))
	_ = gofuncy.Go(send(msg), gofuncy.WithName("sender-b"))
	_ = gofuncy.Go(send(msg), gofuncy.WithName("sender-c"))

	_ = gofuncy.Go(receive(msg), gofuncy.WithName("receiver-a"))
	_ = receive(msg)(context.Background())
	fmt.Println("done")
}

func send(msg chan string) gofuncy.Func {
	return func(ctx context.Context) error {
		for {
			// fmt.Println("sending message")
			msg <- fmt.Sprintf("Hello World #%s", gofuncy.RoutineFromContext(ctx))
			time.Sleep(300 * time.Millisecond)
		}
	}
}

func receive(msg chan string) gofuncy.Func {
	return func(ctx context.Context) error {
		for m := range msg {
			fmt.Println(m, "by", gofuncy.RoutineFromContext(ctx))
			// fmt.Println(m, len(msg))
			time.Sleep(time.Second)
		}
		return nil
	}
}
