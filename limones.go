package main

import (
	"bytes"
	"fmt"
	"github.com/fhs/gompd/mpd"
	"os"
	"os/exec"
	"strconv"
	"strings"
	"time"
)

type item struct {
	Cache  string
	Update updater
}

type updater func(i *item)

func (i *item) start() {
	go i.Update(i)
}

func compose(items map[string]*item) (out string) {
	const sep string = "%{F#ffae81ff} | %{F#ffd1d1d1}"
	return "%{l} %{F#fff4bf75}" +
		items["host"].Cache + sep +
		items["desktop"].Cache + sep +
		items["cpu"].Cache + sep +
		items["memory"].Cache + sep +
		items["battery"].Cache + sep +
		items["sound"].Cache + " %{r} " +
		items["music"].Cache + sep +
		items["date"].Cache + sep +
		items["kernel"].Cache
}

func command(name string, args ...string) string {
	out, _ := exec.Command(name, args...).Output()
	return strings.TrimSpace(string(out))
}

func main() {

	items := make(map[string]*item)

	items["host"] = &item{"", func(i *item) {
		for {
			i.Cache, _ = os.Hostname()
			time.Sleep(time.Second * time.Duration(1000))
		}
	}}
	items["host"].start()
	items["desktop"] = &item{"", func(i *item) {
		for {
			i.Cache = command("bash", "-c", "xprop -root _NET_CURRENT_DESKTOP | awk '{print $3+1}'")
			time.Sleep(time.Second * time.Duration(5))
		}
	}}
	items["desktop"].start()
	items["cpu"] = &item{"", func(i *item) {
		for {
			var buffer bytes.Buffer
			buffer.WriteString("Cpu: ")
			buffer.WriteString(strings.TrimSpace(command("bash", "-c", "echo $[100-$(vmstat 1 2|tail -1|awk '{print $15}')]")))
			buffer.WriteString("% ")
			buffer.WriteString(strings.TrimSpace(command("bash", "-c", "sensors | grep thinkpad-isa-0000 -A 5 | grep temp1 | grep -o '+[0-9]*\\.[0-9]'")))
			buffer.WriteString("C ")
			buffer.WriteString(strings.TrimSpace(command("bash", "-c", "sensors | grep thinkpad-isa-0000 -A 5 | grep fan1 | grep -o '[0-9]* RPM'")))
			i.Cache = buffer.String()
			time.Sleep(time.Second * time.Duration(5))
		}
	}}
	items["cpu"].start()
	items["memory"] = &item{"", func(i *item) {
		for {
			var buffer bytes.Buffer
			buffer.WriteString("Mem: ")
			buffer.WriteString(command("bash", "-c", "free -m | awk 'NR==2{printf \"%.f%%\", $3*100/$2 }'"))
			i.Cache = buffer.String()
			time.Sleep(time.Second * time.Duration(10))

		}
	}}
	items["memory"].start()
	items["battery"] = &item{"", func(i *item) {
		for {
			var buffer bytes.Buffer
			buffer.WriteString("Bat: ")
			buffer.WriteString(command("cat", "/sys/class/power_supply/BAT0/capacity"))
			buffer.WriteString("%")
			i.Cache = buffer.String()
			time.Sleep(time.Second * time.Duration(30))

		}
	}}
	items["battery"].start()
	items["sound"] = &item{"", func(i *item) {
		for {
			var buffer bytes.Buffer
			buffer.WriteString("Snd: ")
			buffer.WriteString(command("bash", "-c", "amixer sget Master | grep -o '[0-9]*\\%'"))
			if _, err := exec.Command("bash", "-c", "amixer sget Master | grep -o '\\[off\\]'").Output(); err == nil {
				buffer.WriteString(" M")
			}
			i.Cache = buffer.String()
			time.Sleep(time.Second * time.Duration(30))
		}
	}}
	items["sound"].start()

	items["music"] = &item{"", func(i *item) {
		i.Cache = "n.a. - n.a."
		client, _ := mpd.Dial("tcp", "localhost:6600")
		for {
			if client == nil || client.Ping() != nil {
				client, _ = mpd.Dial("tcp", "localhost:6600")
				time.Sleep(time.Second * time.Duration(5))
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

			var buffer bytes.Buffer
			buffer.WriteString(artist + " - " + title)
			i.Cache = buffer.String()
			time.Sleep(time.Second * time.Duration(5))
		}
	}}
	items["music"].start()
	items["date"] = &item{"", func(i *item) {
		for {
			t := time.Now().UTC()
			i.Cache = t.Weekday().String() + " " +
				strconv.Itoa(t.Day()) + " " +
				t.Month().String() + " " +
				strconv.Itoa(t.Year()) + " " +
				fmt.Sprintf("%02d", t.Hour()) + ":" +
				fmt.Sprintf("%02d", t.Minute()) + " UTC"

			time.Sleep(time.Second * time.Duration(30))

		}
	}}
	items["date"].start()
	items["kernel"] = &item{"", func(i *item) {
		for {
			i.Cache = command("uname", "-r")
			time.Sleep(time.Second * time.Duration(200))

		}
	}}
	items["kernel"].start()

	time.Sleep(time.Second * time.Duration(2))
	for {
		fmt.Println(compose(items))
		// Sleep 5 seconds
		time.Sleep(time.Second * time.Duration(5))
	}
}
