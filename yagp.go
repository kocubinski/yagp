package main

import (
	"fmt"
	"github.com/gdamore/tcell/v2"
	"github.com/hpcloud/tail"
	"log"
	"os"
	"time"
)

type match struct {
	line string
}

type timer struct {
	startedAt time.Time
	duration  int64
	text      string
	onEnd     func()
}

type appState struct {
	spellTriggers []spellTrigger
	timers        []timer
	spellBook     spellBook
	ui            screenOpts
	casting       *spell
}

type screenOpts struct {
	width  int
	height int
	screen tcell.Screen
}

var haddyLogFile = "/home/mtkoan/Games/everquest/drive_c/eq/Logs/eqlog_Haddy_project1999.txt"
var hadgarLogFile = "/home/mtkoan/Games/everquest/drive_c/eq/Logs/eqlog_Hadgar_P1999Green.txt"

func (state *appState) tailLog(f string) error {
	seekInfo := &tail.SeekInfo{Offset: 0, Whence: 2}

	simLog := os.Getenv("SIM_LOG")
	if simLog != "" {
		f = simLog
		seekInfo = &tail.SeekInfo{Offset: 0, Whence: 0}
	}

	t, err := tail.TailFile(f, tail.Config{Follow: true, Location: seekInfo})
	if err != nil {
		log.Fatalf("%+v", err)
		return err
	}
	for line := range t.Lines {
		state.handeLine(line)
	}
	return nil
}

func (state *appState) handeLine(line *tail.Line) {
	txt := line.Text[27:]
	for _, t := range state.spellTriggers {
		m := t.re.FindStringSubmatch(txt)
		if len(m) > 0 {
			res := t.actor(state, m)
			if res {
				break
			}
		}
	}
}

func (state *appState) draw() {
	y := 0
	for _, t := range state.timers {
		state.ui.drawLine(tcell.StyleDefault, t.text, y)
		y++
	}
	rem := state.ui.height - y
	if rem > 0 {
		for i := 0; i < rem; i++ {
			state.ui.drawLine(tcell.StyleDefault, "", y+i)
		}
	}
}

func (state *appState) updateTimers() {

}

// drawLine draws a line of text on the screen but truncates it if it's too long.
func (so screenOpts) drawLine(style tcell.Style, text string, row int) {
	x := 0
	if row > so.height {
		return
	}

	for _, r := range []rune(text) {
		if x >= so.width {
			break
		}
		so.screen.SetContent(x, row, r, nil, style)
		x++
	}
	remainder := so.width - x
	if remainder > 0 {
		for i := 0; i < remainder; i++ {
			so.screen.SetContent(x+i, row, ' ', nil, style)
		}
	}
}

func drawText(s tcell.Screen, x1, y1, x2, y2 int, style tcell.Style, text string) {
	row := y1
	col := x1
	for _, r := range []rune(text) {
		s.SetContent(col, row, r, nil, style)
		col++
		if col >= x2 {
			row++
			col = x1
		}
		if row > y2 {
			break
		}
	}
}

func drawBox(s tcell.Screen, x1, y1, x2, y2 int, style tcell.Style, text string) {
	if y2 < y1 {
		y1, y2 = y2, y1
	}
	if x2 < x1 {
		x1, x2 = x2, x1
	}

	// Fill background
	for row := y1; row <= y2; row++ {
		for col := x1; col <= x2; col++ {
			s.SetContent(col, row, ' ', nil, style)
		}
	}

	// Draw borders
	for col := x1; col <= x2; col++ {
		s.SetContent(col, y1, tcell.RuneHLine, nil, style)
		s.SetContent(col, y2, tcell.RuneHLine, nil, style)
	}
	for row := y1 + 1; row < y2; row++ {
		s.SetContent(x1, row, tcell.RuneVLine, nil, style)
		s.SetContent(x2, row, tcell.RuneVLine, nil, style)
	}

	// Only draw corners if necessary
	if y1 != y2 && x1 != x2 {
		s.SetContent(x1, y1, tcell.RuneULCorner, nil, style)
		s.SetContent(x2, y1, tcell.RuneURCorner, nil, style)
		s.SetContent(x1, y2, tcell.RuneLLCorner, nil, style)
		s.SetContent(x2, y2, tcell.RuneLRCorner, nil, style)
	}

	drawText(s, x1+1, y1+1, x2-1, y2-1, style, text)
}

type uiTick struct{}

func main() {
	defStyle := tcell.StyleDefault.Background(tcell.ColorReset).Foreground(tcell.ColorReset)
	boxStyle := tcell.StyleDefault.Foreground(tcell.ColorWhite).Background(tcell.ColorPurple)

	// Initialize ui
	s, err := tcell.NewScreen()
	if err != nil {
		log.Fatalf("%+v", err)
	}
	if err := s.Init(); err != nil {
		log.Fatalf("%+v", err)
	}
	s.SetStyle(defStyle)
	s.EnableMouse()
	s.EnablePaste()
	s.Clear()

	// Draw initial boxes
	//drawBox(s, 1, 1, 42, 7, boxStyle, "Click and drag to draw a box")
	//drawBox(s, 5, 9, 32, 14, boxStyle, "Press C to reset")

	quit := func() {
		// You have to catch panics in a defer, clean up, and
		// re-raise them - otherwise your application can
		// die without leaving any diagnostic trace.
		maybePanic := recover()
		s.Fini()
		if maybePanic != nil {
			panic(maybePanic)
		}
	}
	defer quit()

	// Here's how to get the ui size when you need it.
	// xmax, ymax := s.Size()

	// Here's an example of how to inject a keystroke where it will
	// be picked up by the next PollEvent call.  Note that the
	// queue is LIFO, it has a limited length, and PostEvent() can
	// return an error.
	// s.PostEvent(tcell.NewEventKey(tcell.KeyRune, rune('a'), 0))

	sb := newSpellBook()
	state := &appState{
		spellBook:     sb,
		ui:            screenOpts{screen: s, width: 120, height: 24},
		timers:        []timer{},
		spellTriggers: sb.makeSpellTriggers(),
	}

	go func() {
		err := state.tailLog(hadgarLogFile)
		if err != nil {
			log.Fatalf("%+v", err)
		}
	}()

	go func() {
		for {
			state.updateTimers()
			time.Sleep(time.Millisecond * 100)
			err := s.PostEvent(tcell.NewEventInterrupt(uiTick{}))
			if err != nil {
				log.Fatalf("%+v", err)
			}
		}
	}()

	// Event loop
	ox, oy := -1, -1
	for {
		// Update ui
		s.Show()

		// Poll event
		ev := s.PollEvent()

		// Process event
		switch ev := ev.(type) {
		case *tcell.EventInterrupt:
			switch ev.Data().(type) {
			case uiTick:
				state.draw()
			}
		case *tcell.EventResize:
			s.Sync()
		case *tcell.EventKey:
			if ev.Key() == tcell.KeyEscape || ev.Key() == tcell.KeyCtrlC {
				return
			} else if ev.Key() == tcell.KeyCtrlL {
				s.Sync()
			} else if ev.Rune() == 'C' || ev.Rune() == 'c' {
				s.Clear()
			}
		case *tcell.EventMouse:
			x, y := ev.Position()

			switch ev.Buttons() {
			case tcell.Button1, tcell.Button2:
				if ox < 0 {
					ox, oy = x, y // record location when click started
				}

			case tcell.ButtonNone:
				if ox >= 0 {
					label := fmt.Sprintf("%d,%d to %d,%d", ox, oy, x, y)
					drawBox(s, ox, oy, x, y, boxStyle, label)
					ox, oy = -1, -1
				}
			}
		}
	}
}
