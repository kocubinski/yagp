package main

import (
	"fmt"
	"github.com/gdamore/tcell/v2"
	"github.com/hpcloud/tail"
	"log"
	"os"
	"sort"
	"strings"
	"time"
)

type timer struct {
	startedAt time.Time
	duration  time.Duration
	text      string
	self      bool
	onEnd     func()
}

func (t timer) render() string {
	remaining := t.duration - time.Since(t.startedAt)
	return fmt.Sprintf("%s; %s", t.text, remaining.Round(time.Second))
}

func (state *appState) addTimer(t timer) {
	state.timers = append(state.timers, t)
	sort.Slice(state.timers, func(i, j int) bool {
		if state.timers[i].self && !state.timers[j].self {
			return true
		}
		return state.timers[i].text < state.timers[j].text
	})
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
var hypermagicLog = "/home/mtkoan/Games/everquest/drive_c/eq/Logs/eqlog_Hypermagic_P1999Green.txt"

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
	txt := strings.TrimSpace(line.Text[27:])
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
		state.ui.drawLine(tcell.StyleDefault, t.render(), y)
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
	for i, t := range state.timers {
		if time.Since(t.startedAt) > t.duration {
			if t.onEnd != nil {
				t.onEnd()
			}
			state.timers = append(state.timers[:i], state.timers[i+1:]...)
		}
	}
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

type uiTick struct{}

func main() {
	defStyle := tcell.StyleDefault.Background(tcell.ColorReset).Foreground(tcell.ColorReset)

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
		err := state.tailLog(hypermagicLog)
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
	oy := -1
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
			_, y := ev.Position()

			switch ev.Buttons() {
			case tcell.Button2:
				if oy < 0 {
					oy = y // record location when click started
				}

			case tcell.ButtonNone:
				if oy >= 0 {
					if oy < len(state.timers) {
						state.timers = append(state.timers[:oy], state.timers[oy+1:]...)
					}
					oy = -1
				}
			}
		}
	}
}
