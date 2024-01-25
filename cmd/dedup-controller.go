package cmd

import (
	"fmt"
	"log"
	"net/http"

	"github.com/Lanyuqiao/faasd-dedup/pkg/dedup"
	"github.com/spf13/cobra"
)

func makeDedupControllerCmd() *cobra.Command {
	var command = &cobra.Command{
		Use:   "dedup",
		Short: "Run the faasd-dedup-controller",
	}

	command.Flags().String("pull-policy", "Always", `Set to "Always" to force a pull of images upon deployment, or "IfNotPresent" to try to use a cached image.`)

	command.RunE = runDedupE

	return command
}

func runDedupE(cmd *cobra.Command, _ []string) error {
	http.HandleFunc("/receive-lsof", dedup.ReceiveLSOF)
	port := 12345
	log.Printf("Dedup service listening on 0.0.0.0:%d", port)
	err := http.ListenAndServe(fmt.Sprintf(":%d", port), nil)
	if err != nil {
		log.Print("Error when starting dedup controller", err)
	}
	return err
}
