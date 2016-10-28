package ui

import (
	"fmt"
	"io"
	"os"
	"strings"

	"github.com/pkg/term"
)

type UI struct {
	t        *term.Term
	b        [1]byte
	complete func(string) string
}

func New() *UI {
	t, err := term.Open("/dev/tty")
	if err != nil {
		panic(err)
	}

	return &UI{
		t: t,
	}
}

func (ui *UI) Close() error {
	return ui.t.Close()
}

type escState int

const (
	escNone escState = iota
	escEnter
	escCode
)

func (ui *UI) ReadLine(prompt string) (string, error) {
	_, err := os.Stdout.WriteString(prompt)
	if err != nil {
		return "", err
	}
	defer os.Stdout.WriteString("\n")

	err = ui.t.SetRaw()
	if err != nil {
		return "", err
	}
	defer ui.t.Restore()

	line := make([]byte, 0, 32)
	cursor := 0

	var esc escState

	ui.t.Write([]byte("\x1b7")) // save cursor position

	for {
		n, err := ui.t.Read(ui.b[:])
		if err != nil {
			return "", err
		}
		if n == 0 {
			return "", io.EOF
		}

		c := ui.b[0]

		if c > 0x7f { // skip non ascii
			continue
		}

		switch esc {
		case escCode:
			switch c {
			case 'C':
				if cursor < len(line) {
					cursor++
				}
			case 'D':
				if cursor > 0 {
					cursor--
				}
			}
			esc = escNone
		case escEnter:
			if c == '[' {
				esc = escCode
				continue
			}
			esc = escNone
			fallthrough
		case escNone:
			switch c {
			case 0x01: // ^A
				cursor = 0
			case 0x02: // ^B
				if cursor > 0 {
					cursor--
				}
			case 0x04: // ^D
				return "", io.EOF
			case 0x05: // ^E
				cursor = len(line)
			case 0x06: // ^F
				if cursor < len(line) {
					cursor++
				}
			case 0x08, 0x7f: // ^H, DEL
				if len(line) > 0 && cursor > 0 {
					line = append(line[:cursor-1], line[cursor:]...)
					cursor--
				}
			case 0x09: // ^I
				compl := ui.complete(string(line))
				if len(line) < len(compl) {
					line = []byte(compl + " ")
					cursor = len(line)
				}
			case 0x0a, 0x0d: // ^J, ^M
				return string(line), nil
			case 0x0b: // ^K
				if cursor < len(line) {
					line = line[:cursor]
					cursor = len(line)
				}
			case 0x0c: // ^L
				ui.t.Write([]byte("\x1b[2J")) // clear screen
				ui.t.Write([]byte("\x1b[H"))  // move cursor to home
				_, err := os.Stdout.WriteString(prompt)
				if err != nil {
					return "", err
				}
				ui.t.Write([]byte("\x1b7")) // save cursor position
				fallthrough
			case 0x15: // ^U
				cursor = 0
				line = line[:0]
			case 0x1b: // ESC
				esc = escEnter
			default:
				if 0x20 <= c && c <= 0x7e {
					if cursor == len(line) {
						line = append(line, c)
					} else {
						line = append(line, 0)
						copy(line[cursor+1:], line[cursor:])
						line[cursor] = c
					}
					cursor++
				}
			}
		}

		ui.t.Write([]byte("\x1b8"))  // restore cursor position
		ui.t.Write([]byte("\x1b[K")) // clear line from cursor to end
		ui.t.Write(line)
		if i := len(line) - cursor; i > 0 {
			ui.t.Write([]byte(fmt.Sprintf("\x1b[%dD", i))) // move cursor left
		}
	}
}

func (ui *UI) Print(args ...interface{}) {
	ui.fprint(os.Stderr, args)
}

func (ui *UI) PrintErr(args ...interface{}) {
	ui.fprint(os.Stderr, args)
}

func (ui *UI) IsTerminal() bool {
	return false
}

func (ui *UI) SetAutoComplete(complete func(string) string) {
	ui.complete = complete
}

func (ui *UI) fprint(f *os.File, args []interface{}) {
	text := fmt.Sprint(args...)
	if !strings.HasSuffix(text, "\n") {
		text += "\n"
	}
	f.WriteString(text)
}
