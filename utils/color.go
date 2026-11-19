package utils

import (
	"fmt"
	"math/rand"

	"github.com/fatih/color"
	"github.com/mattn/go-colorable"
	"github.com/muesli/termenv"
)

// PrintlnWithColor fix the color of the output on Windows.
func PrintlnWithColor(a ...any) {
	fmt.Fprintln(colorable.NewColorableStdout(), a...)
}

// RandomColor wraps in into a random foreground color.
func RandomColor(in string) string {
	colors := []color.Attribute{
		color.FgRed,
		color.FgGreen,
		color.FgYellow,
		color.FgBlue,
		color.FgMagenta,
		color.FgCyan,
		color.FgWhite,
	}

	index := rand.Intn(len(colors)) // #nosec G404 -- picking a log prefix color does not need a crypto source

	return color.New(colors[index]).Sprintf("%s", in)
}

// ColorFg colors val with the terminal color identified by color.
func ColorFg(val, color string) string {
	return termenv.String(val).Foreground(termenv.ColorProfile().Color(color)).String()
}

// ColorBg sets the background of val to the terminal color identified by color.
func ColorBg(val, color string) string {
	return termenv.String(val).Background(termenv.ColorProfile().Color(color)).String()
}
