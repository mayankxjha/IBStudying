package ibs

import (
	"fmt"
	"time"

	"github.com/gdamore/tcell/v2"
	"github.com/spf13/cobra"
)

var (
	pomoWork      time.Duration
	pomoShort     time.Duration
	pomoLong      time.Duration
	pomoRainTheme string
)

var pomodoroCmd = &cobra.Command{
	Use:     "pomodoro",
	Aliases: []string{"pomo"},
	Short:   "Pomodoro timer rendered over the Matrix rain",
	Long: `A Pomodoro timer painted on top of the Matrix rain animation.

Defaults to the canonical 25 / 5 / 15 minute pattern: 25-minute focus blocks,
5-minute short breaks, and a 15-minute long break after every fourth block.

Controls:
  space     pause / resume
  s         skip the current phase
  r         reset the current phase
  q / Esc   quit`,
	RunE: func(ibs *cobra.Command, args []string) error {
		return runPomodoro()
	},
}

func init() {
	rootCmd.AddCommand(pomodoroCmd)
	pomodoroCmd.Flags().DurationVar(&pomoWork, "work", 25*time.Minute, "duration of a work block")
	pomodoroCmd.Flags().DurationVar(&pomoShort, "short", 5*time.Minute, "duration of a short break")
	pomodoroCmd.Flags().DurationVar(&pomoLong, "long", 15*time.Minute, "duration of a long break")
	pomodoroCmd.Flags().StringVar(&pomoRainTheme, "rain", "green", "rain color theme: green, red, blue, cyan, magenta, rainbow")
}

type pomoPhase int

const (
	phaseWork pomoPhase = iota
	phaseShort
	phaseLong
)

type pomoState struct {
	phase      pomoPhase
	cycleInSet int
	completed  int
	deadline   time.Time
	paused     bool
	pausedAt   time.Duration
}

func newPomoState() *pomoState {
	st := &pomoState{phase: phaseWork, cycleInSet: 1}
	st.deadline = time.Now().Add(st.duration())
	return st
}

func (st *pomoState) duration() time.Duration {
	switch st.phase {
	case phaseWork:
		return pomoWork
	case phaseShort:
		return pomoShort
	case phaseLong:
		return pomoLong
	}
	return 0
}

func (st *pomoState) remaining() time.Duration {
	if st.paused {
		return st.pausedAt
	}
	if r := time.Until(st.deadline); r > 0 {
		return r
	}
	return 0
}

func (st *pomoState) togglePause() {
	if st.paused {
		st.deadline = time.Now().Add(st.pausedAt)
		st.paused = false
	} else {
		st.pausedAt = st.remaining()
		st.paused = true
	}
}

func (st *pomoState) resetPhase() {
	if st.paused {
		st.pausedAt = st.duration()
	} else {
		st.deadline = time.Now().Add(st.duration())
	}
}

func (st *pomoState) advance() {
	switch st.phase {
	case phaseWork:
		st.completed++
		if st.cycleInSet >= 4 {
			st.phase = phaseLong
		} else {
			st.phase = phaseShort
		}
	case phaseShort:
		st.cycleInSet++
		st.phase = phaseWork
	case phaseLong:
		st.cycleInSet = 1
		st.phase = phaseWork
	}
	if st.paused {
		st.pausedAt = st.duration()
	} else {
		st.deadline = time.Now().Add(st.duration())
	}
}

func (st *pomoState) maybeAdvance() {
	if !st.paused && time.Now().After(st.deadline) {
		st.advance()
	}
}

func runPomodoro() error {

	matrixTheme = pomoRainTheme

	s, err := tcell.NewScreen()
	if err != nil {
		return err
	}
	if err := s.Init(); err != nil {
		return err
	}

	quit := make(chan struct{})
	defer s.Fini()
	defer close(quit)

	s.SetStyle(tcell.StyleDefault.Background(tcell.ColorBlack))
	s.Clear()
	s.HideCursor()

	width, height := s.Size()
	drops := make([]*drop, width)
	for i := range drops {
		drops[i] = newDrop(height)
	}

	events := make(chan tcell.Event, 4)
	go func() {
		for {
			ev := s.PollEvent()
			if ev == nil {
				return
			}
			select {
			case events <- ev:
			case <-quit:
				return
			}
		}
	}()

	st := newPomoState()
	ticker := time.NewTicker(50 * time.Millisecond)
	defer ticker.Stop()

	for {
		select {
		case ev := <-events:
			if k, ok := ev.(*tcell.EventKey); ok {
				switch {
				case k.Key() == tcell.KeyEscape,
					k.Key() == tcell.KeyCtrlC,
					k.Rune() == 'q':
					return nil
				case k.Rune() == ' ':
					st.togglePause()
				case k.Rune() == 's':
					st.advance()
				case k.Rune() == 'r':
					st.resetPhase()
				}
			}

		case <-ticker.C:
			if width == 0 || height < 2 {
				continue
			}
			st.maybeAdvance()
			pomoTickRain(s, drops, height)
			drawTimerBox(s, st, width, height)
			s.Show()
		}
	}
}

func pomoTickRain(s tcell.Screen, drops []*drop, height int) {
	for x, d := range drops {
		d.tick++
		if d.tick < d.speed {
			continue
		}
		d.tick = 0
		d.head++

		if intn(5) == 0 {
			d.runes[intn(len(d.runes))] = matrixRunes[intn(len(matrixRunes))]
		}

		clearY := d.head - d.length
		if clearY >= 0 && clearY < height {
			s.SetContent(x, clearY, ' ', nil, tcell.StyleDefault)
		}

		paintDrop(s, x, d, height)

		if d.head-d.length >= height {
			nd := newDrop(height)
			nd.head = -intn(height / 2)
			drops[x] = nd
		}
	}
}

func drawTimerBox(s tcell.Screen, st *pomoState, sw, sh int) {
	const boxW, boxH = 52, 9
	if sw < boxW+2 || sh < boxH+2 {
		return
	}
	bx := (sw - boxW) / 2
	by := (sh - boxH) / 2

	var label string
	var phaseColor tcell.Color
	switch st.phase {
	case phaseWork:
		label, phaseColor = "WORK", tcell.ColorRed
	case phaseShort:
		label, phaseColor = "SHORT BREAK", tcell.ColorLightGreen
	case phaseLong:
		label, phaseColor = "LONG BREAK", tcell.ColorAqua
	}

	bg := tcell.StyleDefault.Background(tcell.ColorBlack)
	border := bg.Foreground(phaseColor).Bold(true)
	labelStyle := bg.Foreground(phaseColor).Bold(true)
	timerStyle := bg.Foreground(tcell.ColorWhite).Bold(true)
	dim := bg.Foreground(tcell.ColorGray)

	for y := by; y < by+boxH; y++ {
		for x := bx; x < bx+boxW; x++ {
			s.SetContent(x, y, ' ', nil, bg)
		}
	}

	for x := bx + 1; x < bx+boxW-1; x++ {
		s.SetContent(x, by, '═', nil, border)
		s.SetContent(x, by+boxH-1, '═', nil, border)
	}
	for y := by + 1; y < by+boxH-1; y++ {
		s.SetContent(bx, y, '║', nil, border)
		s.SetContent(bx+boxW-1, y, '║', nil, border)
	}
	s.SetContent(bx, by, '╔', nil, border)
	s.SetContent(bx+boxW-1, by, '╗', nil, border)
	s.SetContent(bx, by+boxH-1, '╚', nil, border)
	s.SetContent(bx+boxW-1, by+boxH-1, '╝', nil, border)

	rem := st.remaining()
	mins := int(rem / time.Minute)
	secs := int(rem.Seconds()) % 60
	timeStr := fmt.Sprintf("%02d:%02d", mins, secs)
	if st.paused {
		timeStr += "   [PAUSED]"
	}
	cycleStr := fmt.Sprintf("Block %d of 4    ·    completed: %d", st.cycleInSet, st.completed)
	tips := "[space] pause   [s] skip   [r] reset   [q] quit"

	drawCentered(s, by+2, bx, boxW, label, labelStyle)
	drawCentered(s, by+3, bx, boxW, timeStr, timerStyle)

	const barW = 32
	frac := 1.0 - float64(rem)/float64(st.duration())
	if frac < 0 {
		frac = 0
	}
	if frac > 1 {
		frac = 1
	}
	drawProgressBar(s, by+4, bx+(boxW-barW-2)/2, barW, frac,
		bg.Foreground(phaseColor),
		bg.Foreground(tcell.ColorDarkGray))

	drawCentered(s, by+5, bx, boxW, cycleStr, dim)
	drawCentered(s, by+6, bx, boxW, tips, dim)
}

func drawCentered(s tcell.Screen, y, x0, w int, text string, style tcell.Style) {
	rs := []rune(text)
	pad := (w - len(rs)) / 2
	if pad < 0 {
		pad = 0
	}
	for i, r := range rs {
		if pad+i >= w {
			break
		}
		s.SetContent(x0+pad+i, y, r, nil, style)
	}
}

func drawProgressBar(s tcell.Screen, y, x, w int, frac float64, fill, empty tcell.Style) {
	s.SetContent(x, y, '[', nil, empty.Bold(true))
	s.SetContent(x+w+1, y, ']', nil, empty.Bold(true))
	filled := int(frac * float64(w))
	for i := 0; i < w; i++ {
		if i < filled {
			s.SetContent(x+1+i, y, '█', nil, fill)
		} else {
			s.SetContent(x+1+i, y, '░', nil, empty)
		}
	}
}
