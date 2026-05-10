package ibs

import (
	"fmt"
	"os"

	"github.com/fatih/color"
	"github.com/spf13/cobra"
)

var banner = `
╔═══════════════════════════════════════════╗
║      ibstudying — Interactive CLI         ║
╚═══════════════════════════════════════════╝`

var rootCmd = &cobra.Command{
	Use:   "ibs",
	Short: "An Interactive study tool.",
	Long:  color.CyanString(banner) + "\nA Study facilitating tool made with Cobra.",
	Run: func(ibs *cobra.Command, args []string) {
		color.Cyan(banner)
		fmt.Println()
		color.Yellow("👋 Welcome!")
		color.White("  Please use:  %s", color.GreenString("ibs --help"))
	},
}

func Execute() {
	if err := rootCmd.Execute(); err != nil {
		color.Red("✗ %v", err)
		os.Exit(1)
	}
}
