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
	io := make(chan string)
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

	var lastSectorReads, lastSectorWrites uint64
	go func(chan<- string) {
		for {
			rIO := regexp.MustCompile("sda(.)+")
			raw, err := ioutil.ReadFile("/proc/diskstats")
			if err != nil {
				report(err)
			}
			match := rIO.FindString(string(raw))
			IOInfos := strings.Split(strings.TrimSpace(strings.Replace(match, "sda", "", 1)), " ")

			parse, err := strconv.Atoi(IOInfos[2])
			if err != nil {
				report(err)
			}
			reads := uint64(parse)
			parse, err = strconv.Atoi(IOInfos[6])
			if err != nil {
				report(err)
			}
			writes := uint64(parse)

			readsPerSecond := float32(reads-lastSectorReads) * 512 / (5 * 60 * 1024)
			writesPerSecond := float32(writes-lastSectorWrites) * 512 / (5 * 60 * 1024)

			lastSectorReads = reads
			lastSectorWrites = writes
			io <- fmt.Sprintf("IO %.02fR %.02fW kb/s", readsPerSecond, writesPerSecond)
			time.Sleep(5 * time.Second)
		}
	}(io)

	var lastCPUTotal, lastCPUIdle uint64
	go func(chan<- string) {
		for {
			rCPU := regexp.MustCompile("cpu(.)+")
			rThermal := regexp.MustCompile("temperatures\\:(\\s)*(\\d)+")
			rFan := regexp.MustCompile("speed\\:(\\s)*(\\d)+")

			raw, err := ioutil.ReadFile("/proc/stat")
			if err != nil {
				report(err)
			}
			match := rCPU.FindString(string(raw))
			cpuInfos := strings.Split(strings.TrimSpace(strings.Replace(match, "cpu", "", 1)), " ")

			parse, err := strconv.Atoi(cpuInfos[3])
			if err != nil {
				report(err)
			}
			idle := uint64(parse)
			var total uint64
			for _, stat := range cpuInfos {
				parse, err := strconv.Atoi(stat)
				if err != nil {
					report(err)
				}
				total += uint64(parse)
			}
			diffTotal := total - lastCPUTotal
			diffIdle := idle - lastCPUIdle
			usage := (uint64(1000)*(diffTotal-diffIdle)/diffTotal + uint64(5)) / uint64(10)

			lastCPUIdle = idle
			lastCPUTotal = total

			raw, err = ioutil.ReadFile("/sys/devices/system/cpu/cpu0/cpufreq/scaling_max_freq")
			if err != nil {
				report(err)
			}
			frequency, err := strconv.Atoi(strings.TrimSpace(string(raw)))
			if err != nil {
				report(err)
			}

			raw, err = ioutil.ReadFile("/proc/acpi/ibm/thermal")
			if err != nil {
				report(err)
			}
			match = rThermal.FindString(string(raw))
			temperature, err := strconv.Atoi(strings.TrimSpace(strings.Replace(match, "temperatures:", "", 1)))
			if err != nil {
				report(err)
			}

			raw, err = ioutil.ReadFile("/proc/acpi/ibm/fan")
			if err != nil {
				report(err)
			}
			match = rFan.FindString(string(raw))
			fan, err := strconv.Atoi(strings.TrimSpace(strings.Replace(match, "speed:", "", 1)))
			if err != nil {
				report(err)
			}

			cpu <- fmt.Sprintf("C %d%% %d MHZ %d°C %d RPM",
				usage,
				frequency/1000,
				temperature,
				fan)
			time.Sleep(5 * time.Second)
		}
	}(cpu)

	go func(chan<- string) {
		for {
			rTotal := regexp.MustCompile("MemTotal\\:(\\s)*(\\d)+")
			rFree := regexp.MustCompile("MemFree\\:(\\s)*(\\d)+")
			rAvailable := regexp.MustCompile("MemAvailable\\:(\\s)*(\\d)+")

			raw, err := ioutil.ReadFile("/proc/meminfo")
			if err != nil {
				report(err)
			}
			content := string(raw)
			matchTotal := rTotal.FindString(content)
			matchFree := rFree.FindString(content)
			matchAvailable := rAvailable.FindString(content)

			total, err := strconv.Atoi(strings.TrimSpace(strings.Replace(matchTotal, "MemTotal:", "", 1)))
			if err != nil {
				report(err)
			}
			free, err := strconv.Atoi(strings.TrimSpace(strings.Replace(matchFree, "MemFree:", "", 1)))
			if err != nil {
				report(err)
			}
			available, err := strconv.Atoi(strings.TrimSpace(strings.Replace(matchAvailable, "MemAvailable:", "", 1)))
			if err != nil {
				report(err)
			}
			percentage := (float32(total) - (float32(free) + float32(available))) / float32(total) * 100.0
			memory <- fmt.Sprintf("M %.02v%%", percentage)
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
			battery <- fmt.Sprintf("B %v%%", percentage)
			time.Sleep(30 * time.Second)
		}
	}(battery)

	go func(chan<- string) {
		for {
			var mute string
			if _, err := exec.Command("bash", "-c", "amixer sget Master | grep 'Front Left:' | grep -o '\\[off\\]'").Output(); err == nil {
				mute = " M"
			}
			sound <- fmt.Sprintf("S %s%s", command("bash", "-c", "amixer sget Master | grep 'Front Left:' | grep -o '[0-9]*\\%'"), mute)
			time.Sleep(10 * time.Second)
		}
	}(sound)

	go func(chan<- string) {
		for {
			r := regexp.MustCompile("\\d\\d\\.")

			raw, err := ioutil.ReadFile("/proc/net/wireless")
			if err != nil {
				report(err)
			}
			content := string(raw)
			match := r.FindString(content)
			link, err := strconv.Atoi(strings.Replace(match, ".", "", 1))
			if err != nil {
				report(err)
			}
			percentage := float32(link) / 70.0 * 100.0

			content = command("iw", "dev", "wlp2s0", "link")
			r = regexp.MustCompile("SSID: (.)+")
			match = r.FindString(content)
			ssid := strings.Replace(match, "SSID: ", "", 1)

			wifi <- fmt.Sprintf("N %s %.02v%%", ssid, percentage)
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
			date <- fmt.Sprintf("%s %d-%02d-%02d %02d:%02d %s %02d:%02d %s", t.Weekday().String()[:3], t.Year(), t.Month(), t.Day(), utc.Hour(), utc.Minute(), utcZone, t.Hour(), t.Minute(), tZone)
			time.Sleep(30 * time.Second)
		}
	}(date)

	for {
		select {
		case outs["host"] = <-host:
		case outs["desktop"] = <-desktop:
		case outs["io"] = <-io:
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
	fmt.Printf("%s%s %s %s%s %s%s %s%s %s%s%s%s%s%s%s%s%s%s%s%s%s%s%s%s %s",
		leftAdjust, greenBackBlackFront,
		outs["host"], redBackGreenFront, powerline, redBackBlackFront,
		outs["desktop"], blackBackRedFront, powerline, blackBackWhiteFront,
		outs["date"], separatorBlue,
		outs["music"], rightAdjust,
		outs["wifi"], separatorBlue,
		outs["sound"], separatorBlue,
		outs["battery"], separatorBlue,
		outs["io"], separatorBlue,
		outs["memory"], separatorBlue,
		outs["cpu"], "\n")
}

func report(err error) {
	fmt.Fprintf(os.Stderr, "Error occured: %v", err)
}
