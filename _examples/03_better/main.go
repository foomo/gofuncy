package main

import (
	"context"
	"fmt"
	"time"

	"github.com/foomo/gofuncy"
)

func main() {

	msg := gofuncy.NewChannel[string](
		gofuncy.ChannelWithBufferSize[string](10),
	)

	fmt.Println("start")

	_ = gofuncy.Go(send(msg), gofuncy.WithName("sender-a"))
	_ = gofuncy.Go(send(msg), gofuncy.WithName("sender-b"))
	_ = gofuncy.Go(send(msg), gofuncy.WithName("sender-c"))

	_ = gofuncy.Go(receive(msg), gofuncy.WithName("receiver-a"))
	_ = receive(msg)(context.Background())
	fmt.Println("done")
}

func send(msg *gofuncy.Channel[string]) gofuncy.Func {
	return func(ctx context.Context) error {
		for {
			if err := msg.Send(ctx, fmt.Sprintf("Hello World")); err != nil {
				return err
			}
			time.Sleep(300 * time.Millisecond)
		}
	}
}

func receive(msg *gofuncy.Channel[string]) gofuncy.Func {
	return func(ctx context.Context) error {
		for m := range msg.Receive() {
			fmt.Println("Message:", m.Data, "Handler:", gofuncy.RoutineFromContext(ctx), "Sender:", gofuncy.SenderFromContext(m.Context()))
			// fmt.Println(m, len(msg))
			time.Sleep(time.Second)
		}
		return nil
	}
}
