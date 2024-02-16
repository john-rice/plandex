package cmd

import (
	"fmt"
	"log"
	"os"
	"plandex/api"
	"plandex/auth"
	"plandex/fs"
	"plandex/lib"
	"plandex/stream"
	streamtui "plandex/stream_tui"
	"plandex/term"

	"github.com/plandex/plandex/shared"
	"github.com/spf13/cobra"
)

var buildBg bool

var buildCmd = &cobra.Command{
	Use:     "build",
	Aliases: []string{"b"},
	Short:   "Send a prompt for the current plan.",
	// Long:  ``,
	Args: cobra.NoArgs,
	Run:  build,
}

func init() {
	RootCmd.AddCommand(buildCmd)

	buildCmd.Flags().BoolVar(&buildBg, "bg", false, "Execute autonomously in the background")
}

func build(cmd *cobra.Command, args []string) {
	if os.Getenv("OPENAI_API_KEY") == "" {
		term.OutputNoApiKeyMsg()
		os.Exit(1)
	}

	auth.MustResolveAuthWithOrg()
	lib.MustResolveProject()

	if lib.CurrentPlanId == "" {
		fmt.Fprintln(os.Stderr, "No current plan")
		return
	}

	lib.MustCheckOutdatedContextWithOutput()

	projectPaths, _, err := fs.GetProjectPaths()

	if err != nil {
		fmt.Fprintln(os.Stderr, "Error getting project paths:", err)
		return
	}

	apiErr := api.Client.BuildPlan(lib.CurrentPlanId, lib.CurrentBranch, shared.BuildPlanRequest{
		ConnectStream: !buildBg,
		ProjectPaths:  projectPaths,
		ApiKey:        os.Getenv("OPENAI_API_KEY"),
	}, stream.OnStreamPlan)

	if apiErr != nil {
		if apiErr.Msg == shared.NoBuildsErr {
			streamtui.Quit()
			fmt.Println("🤷‍♂️ This plan has no pending changes to build")
			return
		}

		fmt.Fprintln(os.Stderr, "Error building plan:", apiErr.Msg)
		return
	}

	if buildBg {
		fmt.Println("🏗️ Building plan in the background")
	} else {
		go func() {
			err := streamtui.StartStreamUI("", true)

			if err != nil {
				log.Printf("Error starting stream UI: %v\n", err)
				os.Exit(1)
			}

			fmt.Println()
			term.PrintCmds("", "changes", "log")

			os.Exit(0)
		}()

		// Wait for the stream to finish
		select {}
	}
}