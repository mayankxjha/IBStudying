package ibs

import (
	"math/rand"
	"time"

	"github.com/gdamore/tcell/v2"
	"github.com/spf13/cobra"
)

var (
	matrixTheme string
	matrixSpeed int
)

var matrixCmd = &cobra.Command{
	Use:   "matrix",
	Short: "Pure-Go cross-platform Matrix rain animation",
	Long: `A Go-native Matrix-style digital rain. Works on Linux, macOS, and Windows
(Windows 10+ console). Press q, Esc, or Ctrl-C to quit.`,
	RunE: func(ibs *cobra.Command, args []string) error {
		return runMatrix()
	},
}

func init() {
	rootCmd.AddCommand(matrixCmd)
	matrixCmd.Flags().StringVarP(&matrixTheme, "theme", "t", "green",
		"color theme: green, red, blue, cyan, magenta, rainbow")
	matrixCmd.Flags().IntVarP(&matrixSpeed, "speed", "s", 50,
		"frame delay in milliseconds (lower = faster)")
}

var matrixRunes = []rune(
	"ｱｲｳｴｵｶｷｸｹｺｻｼｽｾｿﾀﾁﾂﾃﾄﾅﾆﾇﾈﾉﾊﾋﾌﾍﾎﾏﾐﾑﾒﾓﾔﾕﾖﾗﾘﾙﾚﾛﾜｦﾝ" +
		"0123456789ABCDEFGHIJKLMNOPQRSTUVWXYZ",
)

type drop struct {
	head   int
	length int
	speed  int
	tick   int
	runes  []rune
	headC  tcell.Color
	bodyC  tcell.Color
	tailC  tcell.Color
}

func intn(n int) int {
	if n <= 0 {
		return 0
	}
	return rand.Intn(n)
}

func newDrop(height int) *drop {
	length := 6 + intn(18)
	rs := make([]rune, length)
	for i := range rs {
		rs[i] = matrixRunes[intn(len(matrixRunes))]
	}
	hc, bc, tc := pickColors(matrixTheme)
	return &drop{

		head:   intn(height+length) - length,
		length: length,
		speed:  1 + intn(3),
		runes:  rs,
		headC:  hc,
		bodyC:  bc,
		tailC:  tc,
	}
}

func runMatrix() error {
	for {
		again, err := renderOnce()
		if err != nil || !again {
			return err
		}
	}
}

func renderOnce() (bool, error) {
	s, err := tcell.NewScreen()
	if err != nil {
		return false, err
	}
	if err := s.Init(); err != nil {
		return false, err
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

	ticker := time.NewTicker(time.Duration(matrixSpeed) * time.Millisecond)
	defer ticker.Stop()

	for {
		select {
		case ev := <-events:
			switch ev := ev.(type) {
			case *tcell.EventKey:
				if ev.Key() == tcell.KeyEscape ||
					ev.Key() == tcell.KeyCtrlC ||
					ev.Rune() == 'q' {
					return false, nil
				}
			case *tcell.EventResize:

				nw, nh := ev.Size()
				if nw == width && nh == height {
					break
				}
				return true, nil
			}

		case <-ticker.C:
			if width == 0 || height < 2 {
				continue
			}
			for x, d := range drops {
				d.tick++
				if d.tick < d.speed {
					continue
				}
				d.tick = 0
				d.head++

				if intn(5) == 0 {
					d.runes[intn(len(d.runes))] =
						matrixRunes[intn(len(matrixRunes))]
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
			s.Show()
		}
	}
}

func paintDrop(s tcell.Screen, x int, d *drop, height int) {
	for i := 0; i < d.length; i++ {
		y := d.head - i
		if y < 0 || y >= height {
			continue
		}
		var style tcell.Style
		switch {
		case i == 0:
			style = tcell.StyleDefault.Foreground(d.headC).Bold(true)
		case i < d.length/3:
			style = tcell.StyleDefault.Foreground(d.bodyC)
		default:
			style = tcell.StyleDefault.Foreground(d.tailC)
		}
		s.SetContent(x, y, d.runes[i], nil, style)
	}
}

func pickColors(theme string) (tcell.Color, tcell.Color, tcell.Color) {
	switch theme {
	case "red":
		return tcell.ColorWhite, tcell.ColorRed, tcell.ColorDarkRed
	case "blue":
		return tcell.ColorWhite, tcell.ColorLightSkyBlue, tcell.ColorBlue
	case "cyan":
		return tcell.ColorWhite, tcell.ColorAqua, tcell.ColorTeal
	case "magenta":
		return tcell.ColorWhite, tcell.ColorFuchsia, tcell.ColorPurple
	case "rainbow":
		palette := []tcell.Color{
			tcell.ColorRed, tcell.ColorOrange, tcell.ColorYellow,
			tcell.ColorLime, tcell.ColorAqua, tcell.ColorFuchsia,
		}
		c := palette[intn(len(palette))]
		return tcell.ColorWhite, c, c
	default:
		return tcell.ColorWhite, tcell.ColorLightGreen, tcell.ColorGreen
	}
}
