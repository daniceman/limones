package main

import (
	"fmt"
	"os"
	"os/exec"
	"strings"
	"time"

	"github.com/fhs/gompd/mpd"
)

const (
	leftAdjust string = "%{l}"
	rightAdjust string = "%{r}"
	green string = "%{F#ffa6e22e}"
	red string = "%{F#fff92672}"
	sepBlue string = " %{F#ff66d9ef}|%{F#fff8f8f2} "
)

func main() {
	host := make(chan string, 10)
	desktop := make(chan string, 10)
	cpu := make(chan string, 10)
	memory := make(chan string, 10)
	battery := make(chan string, 10)
	sound := make(chan string, 10)
	wifi := make(chan string, 10)
	music := make(chan string, 10)
	date := make(chan string, 10)
	kernel := make(chan string, 10)

	outs := make(map[string]string)

	go func(chan<- string) {
		for {
			h, _ := os.Hostname()
			host <- h
			time.Sleep(1000 * time.Second)
		}
	}(host)

	go func(chan<- string) {
		for {
			desktop <- command("bspc", "query", "-D", "-d", "--names")
			time.Sleep(5 * time.Second)
		}
	}(desktop)

	go func(chan<- string) {
		for {
			cpu <- fmt.Sprintf("C: %s%% %s MHZ %s °C %s",
				command("bash", "-c", "echo $[100-$(vmstat 1 2|tail -1|awk '{print $15}')]"),
				command("bash", "-c", "cat /sys/devices/system/cpu/cpu0/cpufreq/scaling_max_freq | awk '{print $1/1000}'"),
				command("bash", "-c", "sensors | grep thinkpad-isa-0000 -A 5 | grep temp1 | grep -o '+[0-9]*\\.[0-9]'"),
				command("bash", "-c", "sensors | grep thinkpad-isa-0000 -A 5 | grep fan1 | grep -o '[0-9]* RPM'"))
			time.Sleep(5 * time.Second)
		}
	}(cpu)

	go func(chan<- string) {
		for {
			memory <- fmt.Sprintf("M: %s", command("bash", "-c", "free -m | awk 'NR==2{printf \"%.f%%\", $3*100/$2 }'"))
			time.Sleep( 10 * time.Second)
		}
	}(memory)

	go func(chan<- string) {
		for {
			battery <- fmt.Sprintf("B: %s%%", command("cat", "/sys/class/power_supply/BAT0/capacity"))
			time.Sleep(30 * time.Second)
		}
	}(battery)

	go func(chan<- string) {
		for {
			var mute string
			if _, err := exec.Command("bash", "-c", "amixer sget Master | grep -o '\\[off\\]'").Output(); err == nil {
				mute = " M"
			}
			sound <- fmt.Sprintf("S: %s%s", command("bash", "-c", "amixer sget Master | grep -o '[0-9]*\\%'"), mute)
			time.Sleep(10 * time.Second)
		}
	}(sound)

	go func(chan<- string) {
		for {
			wifi <- fmt.Sprintf("N: %s", command("bash", "-c", "iw dev wlp2s0 link | grep -o 'SSID:.*' | cut -c7-"))
			time.Sleep(30 * time.Second)
		}
	}(wifi)

	go func(chan<- string) {
		music <- "n.a. - n.a."
		client, _ := mpd.Dial("tcp", "localhost:6600")
		for {
			if client == nil || client.Ping() != nil {
				client, _ = mpd.Dial("tcp", "localhost:6600")
				time.Sleep(5 * time.Second)
				continue
			}
			defer client.Close()

			attrs, err := client.CurrentSong()
			if err != nil {
				continue
			}

			artist := attrs["Artist"]
			if artist == "" {
				artist = "n.a."
			}
			title := attrs["Title"]
			if title == "" {
				title = "n.a."
			}

			music <- fmt.Sprintf("%s - %s", artist, title)
			time.Sleep(10 * time.Second)
		}
	}(music)

	go func(chan<- string) {
		for {
			t := time.Now().UTC()
			date <- fmt.Sprintf("%s %d %s %d %02d:%02d UTC", t.Weekday().String(), t.Day(), t.Month().String(), t.Year(), t.Hour(), t.Minute())
			time.Sleep(30 * time.Second)
		}
	}(date)

	go func(chan<- string) {
		for {
			kernel <- command("uname", "-r")
			time.Sleep(1000 * time.Second)
		}
	}(kernel)

	for {
		select {
		case outs["host"] = <-host:
		case outs["desktop"] = <-desktop:
		case outs["cpu"] = <-cpu:
		case outs["memory"] = <-memory:
		case outs["battery"] = <-battery:
		case outs["sound"] = <-sound:
		case outs["wifi"] = <-wifi:
		case outs["music"] = <-music:
		case outs["date"] = <-date:
		case outs["kernel"] = <-kernel:
		}
		print(outs)
	}
}

func command(name string, args ...string) string {
	// In this case it is really ok to drop err's
	out, _ := exec.Command(name, args...).Output()
	return strings.TrimSpace(string(out))
}

func print(outs map[string]string) {
	fmt.Printf("%s %s%s%s%s%s%s%s%s%s%s%s%s%s%s%s%s%s%s%s%s%s %s",
		leftAdjust,
		green,
		outs["host"], sepBlue,
		red,
		outs["desktop"], sepBlue,
		outs["cpu"], sepBlue,
		outs["memory"], sepBlue,
		outs["battery"], sepBlue,
		outs["sound"], sepBlue,
		outs["wifi"], rightAdjust,
		outs["music"], sepBlue,
		outs["date"], sepBlue,
		outs["kernel"], "\n")
}
