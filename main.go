package main

import (
	"fmt"
	"os"
	"path/filepath"
	"time"

	"codeberg.org/asartalo/assg/internal/commands"
	"github.com/spf13/cobra"
)

var rootCmd = &cobra.Command{
	Use:   "assg",
	Short: "ASSG is Asartalo’s Static Site Generator",
	Long: "ASSG (Asartalo’s Static Site Generator) is a static site generator custom\n" +
		"built for Asartalo’s website at https://brainchildprojects.com and for other\n" +
		"projects.\n\n" +
		"Visit https://codeberg.org/asartalo/assg for the source and for more information.",
	Run: func(cmd *cobra.Command, args []string) {
		cmd.Help()
	},
}

var buildCmd = &cobra.Command{
	Use:   "build",
	Short: "Build the static site",
	Long:  `Generates the static site based on the content and templates.`,
	Run: func(cmd *cobra.Command, args []string) {
		fmt.Println("Building site...")
		srcDir, err := os.Getwd()
		if err != nil {
			fmt.Println("Error:", err)
			return
		}

		outputDir := filepath.Join(srcDir, "public")
		err = commands.Build(srcDir, outputDir, false, verbose, time.Now())
		if err != nil {
			fmt.Println("Error:", err)
			return
		}
	},
}

var serveCmd = &cobra.Command{
	Use:   "serve",
	Short: "Serve the static site",
	Long:  `Starts a local server to preview the static site.`,
	Run: func(cmd *cobra.Command, args []string) {
		srcDir, err := os.Getwd()
		if err != nil {
			fmt.Println("Error:", err)
			return
		}

		err = commands.Serve(srcDir, false)
		if err != nil {
			fmt.Println("Error:", err)
			return
		}
	},
}

var includeDrafts bool
var verbose bool

func init() {
	rootCmd.AddCommand(buildCmd)
	rootCmd.AddCommand(serveCmd)

	// Add flags
	buildCmd.Flags().BoolVarP(&verbose, "verbose", "v", false, "Print verbose output")
	serveCmd.Flags().BoolVar(&includeDrafts, "include-drafts", false, "Include draft pages when serving")
}

func main() {
	if err := rootCmd.Execute(); err != nil {
		fmt.Println(err)
		os.Exit(1)
	}
}
