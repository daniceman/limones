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
	wifi := make(chan string, 10)
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

	go func(chan<- string) {
		for {
			cpu <- fmt.Sprintf("Cpu: %s%% %s MHZ %s °C %s",
				command("bash", "-c", "echo $[100-$(vmstat 1 2|tail -1|awk '{print $15}')]"),
				command("bash", "-c", "cat /sys/devices/system/cpu/cpu0/cpufreq/scaling_max_freq | awk '{print $1/1000}'"),
				command("bash", "-c", "sensors | grep thinkpad-isa-0000 -A 5 | grep temp1 | grep -o '+[0-9]*\\.[0-9]'"),
				command("bash", "-c", "sensors | grep thinkpad-isa-0000 -A 5 | grep fan1 | grep -o '[0-9]* RPM'"))
			time.Sleep(time.Second * time.Duration(5))
		}
	}(cpu)

	go func(chan<- string) {
		for {
			memory <- fmt.Sprintf("Mem: %s", command("bash", "-c", "free -m | awk 'NR==2{printf \"%.f%%\", $3*100/$2 }'"))
			time.Sleep(time.Second * time.Duration(10))
		}
	}(memory)

	go func(chan<- string) {
		for {
			battery <- fmt.Sprintf("Bat: %s%%", command("cat", "/sys/class/power_supply/BAT0/capacity"))
			time.Sleep(time.Second * time.Duration(30))
		}
	}(battery)

	go func(chan<- string) {
		for {
			var mute string
			if _, err := exec.Command("bash", "-c", "amixer sget Master | grep -o '\\[off\\]'").Output(); err == nil {
				mute = " M"
			}
			sound <- fmt.Sprintf("Snd: %s%s", command("bash", "-c", "amixer sget Master | grep -o '[0-9]*\\%'"), mute)
			time.Sleep(time.Second * time.Duration(30))
		}
	}(sound)

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
	const sep string = "%{F#ff66d9ef} | %{F#fff8f8f2}"
	const start string = "%{l} %{F#ffa6e22e}"
	const rightAdjust string = "%{r}"
	fmt.Printf("%s %s %s %s %s %s %s %s %s %s %s %s %s %s %s %s %s %s %s %s",
		start,
		outs["host"], sep,
		outs["desktop"], sep,
		outs["cpu"], sep,
		outs["memory"], sep,
		outs["battery"], sep,
		outs["sound"], sep,
		outs["wifi"], rightAdjust,
		outs["music"], sep,
		outs["date"], sep,
		outs["kernel"])
}
