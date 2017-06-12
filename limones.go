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

type Item struct {
	Cache  string
	Update Updater
}

type Updater func(i *Item)

func (i *Item) Start() {
	go i.Update(i)
}

func Compose(items map[string]*Item) (out string) {
	const sep string = " | "
	return "%{l} " + items["host"].Cache + sep + items["desktop"].Cache + sep + items["cpu"].Cache + sep + items["memory"].Cache + sep + items["battery"].Cache + " %{r} " + items["music"].Cache + sep + items["date"].Cache + sep + items["kernel"].Cache
}

func Command(name string, args ...string) string {
	out, _ := exec.Command(name, args...).Output()
	return strings.TrimSpace(string(out))
}

func main() {

	items := make(map[string]*Item)

	items["host"] = &Item{"", func(i *Item) {
		for {
			i.Cache, _ = os.Hostname()
			time.Sleep(time.Second * time.Duration(1000))
		}
	}}
	items["host"].Start()
	items["desktop"] = &Item{"", func(i *Item) {
		for {
			i.Cache = Command("bash", "-c", "xprop -root _NET_CURRENT_DESKTOP | awk '{print $3+1}'")
			time.Sleep(time.Second * time.Duration(5))
		}
	}}
	items["desktop"].Start()
	items["cpu"] = &Item{"", func(i *Item) {
		for {
			var buffer bytes.Buffer
			buffer.WriteString("Cpu: ")
			buffer.WriteString(strings.TrimSpace(Command("bash", "-c", "echo $[100-$(vmstat 1 2|tail -1|awk '{print $15}')]")))
			buffer.WriteString("% ")
			buffer.WriteString(strings.TrimSpace(Command("bash", "-c", "sensors | grep thinkpad-isa-0000 -A 5 | grep temp1 | grep -o '+[0-9]*\\.[0-9]'")))
			buffer.WriteString("C ")
			buffer.WriteString(strings.TrimSpace(Command("bash", "-c", "sensors | grep thinkpad-isa-0000 -A 5 | grep fan1 | grep -o '[0-9]* RPM'")))
			i.Cache = buffer.String()
			time.Sleep(time.Second * time.Duration(5))
		}
	}}
	items["cpu"].Start()
	items["memory"] = &Item{"", func(i *Item) {
		for {
			var buffer bytes.Buffer
			buffer.WriteString("Mem: ")
			buffer.WriteString(Command("bash", "-c", "free -m | awk 'NR==2{printf \"%.f%%\", $3*100/$2 }'"))
			i.Cache = buffer.String()
			time.Sleep(time.Second * time.Duration(10))

		}
	}}
	items["memory"].Start()
	items["battery"] = &Item{"", func(i *Item) {
		for {
			var buffer bytes.Buffer
			buffer.WriteString("Bat: ")
			buffer.WriteString(Command("cat", "/sys/class/power_supply/BAT0/capacity"))
			buffer.WriteString("%")
			i.Cache = buffer.String()
			time.Sleep(time.Second * time.Duration(30))

		}
	}}
	items["battery"].Start()

	items["music"] = &Item{"", func(i *Item) {
		for {
			client, err := mpd.Dial("tcp", "localhost:6600")
			if err != nil {
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
	items["music"].Start()
	items["date"] = &Item{"", func(i *Item) {
		for {
			t := time.Now().UTC()
			i.Cache = t.Weekday().String() + " " + strconv.Itoa(t.Day()) + " " + t.Month().String() + " " + strconv.Itoa(t.Year()) + " " + fmt.Sprintf("%02d", t.Hour()) + ":" + fmt.Sprintf("%02d", t.Minute()) + " UTC"

			time.Sleep(time.Second * time.Duration(30))

		}
	}}
	items["date"].Start()
	items["kernel"] = &Item{"", func(i *Item) {
		for {
			i.Cache = Command("uname", "-r")
			time.Sleep(time.Second * time.Duration(200))

		}
	}}
	items["kernel"].Start()

	time.Sleep(time.Second * time.Duration(2))
	for {
		fmt.Println(Compose(items))
		// Sleep a second
		time.Sleep(time.Second * time.Duration(5))
	}
}
