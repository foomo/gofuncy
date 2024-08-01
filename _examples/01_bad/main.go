package main

import (
	"fmt"
	"time"
)

func main() {

	msg := make(chan string, 10)

	fmt.Println("start")
	go send(msg)
	go send(msg)
	go send(msg)

	go receive("a", msg)
	receive("b", msg)
	fmt.Println("done")
}

func send(msg chan string) {
	var i int
	for {
		// fmt.Println("sending message")
		msg <- fmt.Sprintf("Hello World #%d", i)
		time.Sleep(300 * time.Millisecond)
		i++
	}
}

func receive(n string, msg chan string) {
	for m := range msg {
		// fmt.Println(m)
		fmt.Println(n, m, len(msg))
		time.Sleep(time.Second)
	}
}
