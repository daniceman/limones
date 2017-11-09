package main

import (
	"fmt"
	"io/ioutil"
	"os"
	"os/exec"
	"regexp"
	"strconv"
	"strings"
	"time"

	"github.com/fhs/gompd/mpd"
)

const (
	leftAdjust          string = "%{l}"
	rightAdjust         string = "%{r}"
	greenBackBlackFront string = "%{B#ffa6e22e}%{F#ff272822}"
	redBackGreenFront   string = "%{B#fff92672}%{F#ffa6e22e}"
	redBackBlackFront   string = "%{B#fff92672}%{F#ff272822}"
	blackBackWhiteFront string = "%{B#ff272822}%{F#fff8f8f2}"
	blackBackRedFront   string = "%{B#ff272822}%{F#fff92672}"
	separatorBlue       string = " %{F#ff66d9ef}|%{F#fff8f8f2} "
	powerline           string = ""
)

func main() {
	host := make(chan string)
	desktop := make(chan string)
	cpu := make(chan string)
	memory := make(chan string)
	battery := make(chan string)
	sound := make(chan string)
	wifi := make(chan string)
	music := make(chan string)
	date := make(chan string)

	outs := make(map[string]string)

	go func(chan<- string) {
		for {
			h, err := os.Hostname()
			if err != nil {
				report(err)
			}
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
			raw, err := ioutil.ReadFile("/proc/meminfo")
			if err != nil {
				report(err)
			}
			content := string(raw)
			rTotal := regexp.MustCompile("MemTotal\\:(\\s)*(\\d)+")
			rActive := regexp.MustCompile("Active\\(anon\\)\\:(\\s)*(\\d)+")
			matchTotal := rTotal.FindString(content)
			matchActive := rActive.FindString(content)

			total, err := strconv.Atoi(strings.TrimSpace(strings.Replace(matchTotal, "MemTotal:", "", 1)))
			if err != nil {
				report(err)
			}
			active, err := strconv.Atoi(strings.TrimSpace(strings.Replace(matchActive, "Active(anon):", "", 1)))
			if err != nil {
				report(err)
			}
			percentage := float32(active) / float32(total) * 100.0
			memory <- fmt.Sprintf("M: %.02v%%", percentage)
			time.Sleep(10 * time.Second)
		}
	}(memory)

	go func(chan<- string) {
		for {
			raw, err := ioutil.ReadFile("/sys/class/power_supply/BAT0/capacity")
			if err != nil {
				report(err)
			}
			percentage, err := strconv.Atoi(strings.TrimSpace(string(raw)))
			if err != nil {
				report(err)
			}
			battery <- fmt.Sprintf("B: %v%%", percentage)
			time.Sleep(30 * time.Second)
		}
	}(battery)

	go func(chan<- string) {
		for {
			var mute string
			if _, err := exec.Command("bash", "-c", "amixer sget Master | grep 'Front Left:' | grep -o '\\[off\\]'").Output(); err == nil {
				mute = " M"
			}
			sound <- fmt.Sprintf("S: %s%s", command("bash", "-c", "amixer sget Master | grep 'Front Left:' | grep -o '[0-9]*\\%'"), mute)
			time.Sleep(10 * time.Second)
		}
	}(sound)

	go func(chan<- string) {
		for {
			raw, err := ioutil.ReadFile("/proc/net/wireless")
			if err != nil {
				report(err)
			}
			content := string(raw)
			r := regexp.MustCompile("\\d\\d\\.")
			match := r.FindString(content)
			link, err := strconv.Atoi(strings.Replace(match, ".", "", 1))
			if err != nil {
				report(err)
			}
			percentage := float32(link) / 70.0 * 100.0

			content = command("bash", "-c", "iw dev wlp2s0 link")
			r = regexp.MustCompile("SSID: (.)+")
			match = r.FindString(content)
			ssid := strings.Replace(match, "SSID: ", "", 1)

			wifi <- fmt.Sprintf("N: %s %.02v%%", ssid, percentage)
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
			utc := time.Now().UTC()
			utcZone, _ := utc.Zone()
			t := time.Now()
			tZone, _ := t.Zone()
			date <- fmt.Sprintf("%s %d-%02d-%02d %02d:%02d %s / %02d:%02d %s", t.Weekday().String()[:3], t.Year(), t.Month(), t.Day(), t.Hour(), t.Minute(), tZone, utc.Hour(), utc.Minute(), utcZone)
			time.Sleep(30 * time.Second)
		}
	}(date)

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
	fmt.Printf("%s%s %s %s%s %s%s %s%s %s%s%s%s%s%s%s%s%s%s%s%s%s%s %s",
		leftAdjust, greenBackBlackFront,
		outs["host"], redBackGreenFront, powerline, redBackBlackFront,
		outs["desktop"], blackBackRedFront, powerline, blackBackWhiteFront,
		outs["date"], separatorBlue,
		outs["music"], rightAdjust,
		outs["wifi"], separatorBlue,
		outs["sound"], separatorBlue,
		outs["battery"], separatorBlue,
		outs["memory"], separatorBlue,
		outs["cpu"], "\n")
}

func report(err error) {
	fmt.Fprintf(os.Stderr, "Error occured: %v", err)
}
