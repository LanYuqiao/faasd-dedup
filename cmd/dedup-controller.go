package cmd

import (
	"fmt"
	"log"
	"net/http"

	"github.com/spf13/cobra"
)

func makeDedupControllerCmd() *cobra.Command {
	var command = &cobra.Command{
		Use:   "provider",
		Short: "Run the faasd-provider",
	}

	command.Flags().String("pull-policy", "Always", `Set to "Always" to force a pull of images upon deployment, or "IfNotPresent" to try to use a cached image.`)

	command.RunE = runDedupE

	return command
}

func runDedupE(cmd *cobra.Command, _ []string) error {
	http.HandleFunc("/receive-lsof", dedup.receiveLSOF)
	port := 12345
	err := http.ListenAndServe(fmt.Sprintf(":%d", port), nil)
	if err != nil {
		log.Print("Error when starting dedup controller", err)
	}
	return err
}
