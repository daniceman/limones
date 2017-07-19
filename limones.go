package main

import (
	"fmt"
	"os"
	"os/exec"
	"strings"
	"time"
)

func main() {

	host := make(chan string, 10)
	desktop := make(chan string, 10)
	cpu := make(chan string, 10)
	memory := make(chan string, 10)
	battery := make(chan string, 10)
	sound := make(chan string, 10)
	music := make(chan string, 10)
	date := make(chan string, 10)
	kernel := make(chan string, 10)

	outs := make(map[string]string)

	go func(chan<- string) {
		for {
			h, _ := os.Hostname()
			host <- h
			time.Sleep(time.Second * time.Duration(1000))
		}
	}(host)

	go func(chan<- string) {
		for {
			desktop <- command("bash", "-c", "xprop -root _NET_CURRENT_DESKTOP | awk '{print $3+1}'")
			time.Sleep(time.Second * time.Duration(1000))
		}
	}(desktop)

	for {
		select {
			case outs["host"] = <- host:
				print(outs)
			case outs["desktop"] = <- desktop:
				print(outs)
		}
	}
}

func command(name string, args ...string) string {
	// In this case it is really ok to drop err's
	out, _ := exec.Command(name, args...).Output()
	return strings.TrimSpace(string(out))
}

func print(outs map[string]string) {
	const sep string = "%{F#ff66d9ef} | %{F#fff8f8f2}"
	const start string = "%{l} %{F#ffa6e22e}"
	fmt.Printf("%s %s %s", start, outs["host"], sep)
}
