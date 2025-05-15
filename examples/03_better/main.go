package main

import (
	"context"
	"fmt"
	"time"

	"github.com/foomo/gofuncy"
	"go.uber.org/zap"
)

func main() {
	l, _ := zap.NewDevelopment()
	ctx, cancel := context.WithTimeout(context.Background(), time.Second)
	ctx = gofuncy.Ctx(ctx).Root()
	defer cancel()

	ch := gofuncy.NewChan[string](
		gofuncy.ChanWithLogger[string](l),
		gofuncy.ChanWithBuffer[string](3),
	)
	defer ch.Close()

	fmt.Println("start")

	_ = gofuncy.Go(ctx, send(ch), gofuncy.WithName("sender-a"))
	_ = gofuncy.Go(ctx, send(ch), gofuncy.WithName("sender-b"))
	_ = gofuncy.Go(ctx, send(ch), gofuncy.WithName("sender-c"))

	_ = gofuncy.Go(ctx, receive(ch), gofuncy.WithName("receiver-a"))
	_ = receive(ch)(ctx)

	fmt.Println("done")
}

func send(ch *gofuncy.Chan[string]) gofuncy.Func {
	return func(ctx context.Context) error {
		name := gofuncy.Ctx(ctx).Name()
		for i := range 3 {
			if err := ch.Send(ctx, fmt.Sprintf("#%d from %s", i, name)); err != nil {
				return err
			}
		}
		fmt.Println("sent all messages from " + name)
		return nil
	}
}

func receive(ch *gofuncy.Chan[string]) gofuncy.Func {
	return func(ctx context.Context) error {
		for {
			select {
			case <-ctx.Done():
				fmt.Println("context done")
				return ctx.Err()
			case m, ok := <-ch.Receive(ctx):
				if ok {
					fmt.Println("Handler:", gofuncy.NameFromContext(ctx), "Message:", m)
				} else {
					fmt.Println("channel closed")
					return nil
				}
			}
		}
	}
}
