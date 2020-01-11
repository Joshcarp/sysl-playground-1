package main

import (
	"errors"
	"fmt"
	"net/http"
	"net/url"
	// "regexp"
	"syscall/js"
	// "encoding/json"
	"github.com/Joshcarp/sysl_testing/pkg/command"
	"github.com/gopherjs/vecty"
	"github.com/gopherjs/vecty/elem"
	b64 "encoding/base64"
	"github.com/gopherjs/vecty/event"
	"github.com/sirupsen/logrus"
	"github.com/spf13/afero"
)

var mychan = make(chan string, 10000)
var mGlobal *Markdown
var info *http.Response

func loadQueryParams() (url.Values, bool) {
	href := js.Global().Get("location").Get("href")
	str := fmt.Sprintf("%s", href)
	u, err := url.Parse(str)
	check(err)
	if len(u.Query()) ==0{
		return u.Query(), false
	}
	return u.Query(), true
}

func encode(str string)string{
	return b64.StdEncoding.EncodeToString([]byte(str))   
}

func decode(str string)string{
	this, _ := b64.StdEncoding.DecodeString(str)
	return string(this)
}
func decodeQueryParams(in url.Values)(string, string){
	foo := decode(in.Get("input"))
	bar :=  decode(in.Get("cmd"))
	return foo,bar
}
func setup()(string, string){
	href, _ := loadQueryParams()
	input, cmd := decodeQueryParams(href)
	fmt.Println("command", cmd)
	if cmd == ""{
	input = `MobileApp:
	Login:
			Server <- Login
	!type LoginData:
			username <: string
			password <: string
	!type LoginResponse:
			message <: string
Server:
	Login(data <: MobileApp.LoginData):
			return MobileApp.LoginResponse`
	cmd = "sysl sd -o \"project.svg\" -s \"MobileApp <- Login\" tmp.sysl"
	}
	fmt.Println(cmd, input)
	fmt.Println("2")
	return input, cmd
}
func main() {
	input, cmd := setup()

	vecty.SetTitle("sysl playground")
	
	vecty.RenderBody(&PageView{
		Input:   input,
		Command: cmd,
	})
	fmt.Println("5")

	// go keepAlive()
}

// PageView is our main page component.
type PageView struct {
	vecty.Core
	Input   string
	Command string
}

// Render implements the vecty.Component interface.
func (p *PageView) Render() vecty.ComponentOrHTML {
	return elem.Body(
		// Display a textarea on the right-hand side of the page.
		elem.Div(
			elem.TextArea(
				vecty.Markup(
					vecty.Style("font-family", "monospace"),
					vecty.Property("rows", 14),
					vecty.Property("cols", 70),

					// When input is typed into the textarea, update the local
					// component state and rerender.
					event.Input(func(e *vecty.Event) {
						p.Input = e.Target.Get("value").String()
						vecty.Rerender(p)
					}),
				),
				vecty.Text(p.Input), // initial textarea text.
			),
			elem.TextArea(
				vecty.Markup(
					vecty.Style("font-family", "monospace"),
					vecty.Property("rows", 1),
					vecty.Property("cols", 70),

					// When input is typed into the textarea, update the local
					// component state and rerender.
					event.Input(func(e *vecty.Event) {
						p.Command = e.Target.Get("value").String()
						vecty.Rerender(p)
					}),
				),
				vecty.Text(p.Command), // initial textarea text.
			),
			vecty.Markup(
				vecty.Style("float", "left"),
			),
		),

		// Render the markdown.
		&Markdown{Input: p.Input, Command: p.Command},
	)
}

// Markdown is a simple component which renders the Input markdown as sanitized
// HTML into a div.
type Markdown struct {
	vecty.Core
	Input   string `vecty:"prop"`
	Command string `vecty:"prop"`
}

// Render implements the vecty.Component interface.
func (m *Markdown) Render() (res vecty.ComponentOrHTML) {
	defer func() {
		if r := recover(); r != nil {
			res = elem.Div(
				vecty.Markup(
					vecty.UnsafeHTML(fmt.Sprintf("%s", r)),
				),
			)
		}
	}()
	fs := afero.NewMemMapFs()
	f, err := fs.Create("/tmp.sysl")
	check(err)

	_, e := f.Write([]byte(m.Input))
	check(e)

	var logger = logrus.New()
	args, err := parseCommandLine(m.Command)
	check(err)
	fmt.Println(args, len(args))
	command.Main2(args, fs, logger, command.Main3)

	this, err := afero.ReadFile(fs, "project.svg")
	check(err)

	foo := fmt.Sprintf("<img src=\"%s\">", string(this))
	fmt.Println(foo)
	return elem.Div(
		vecty.Markup(
			vecty.UnsafeHTML(
				foo),
		),
	)
}

func check(err error) {
	if err != nil {
		panic(err)
	}
}

var signal = make(chan int)

func parseCommandLine(command string) ([]string, error) {
	var args []string
	state := "start"
	current := ""
	quote := "\""
	escapeNext := true
	for i := 0; i < len(command); i++ {
		c := command[i]

		if state == "quotes" {
			if string(c) != quote {
				current += string(c)
			} else {
				args = append(args, current)
				current = ""
				state = "start"
			}
			continue
		}

		if escapeNext {
			current += string(c)
			escapeNext = false
			continue
		}

		if c == '\\' {
			escapeNext = true
			continue
		}

		if c == '"' || c == '\'' {
			state = "quotes"
			quote = string(c)
			continue
		}

		if state == "arg" {
			if c == ' ' || c == '\t' {
				args = append(args, current)
				current = ""
				state = "start"
			} else {
				current += string(c)
			}
			continue
		}

		if c != ' ' && c != '\t' {
			state = "arg"
			current += string(c)
		}
	}

	if state == "quotes" {
		return []string{}, errors.New(fmt.Sprintf("Unclosed quote in command line: %s", command))
	}

	if current != "" {
		args = append(args, current)
	}

	return args, nil
}