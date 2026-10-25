package cli

import (
	"context"
	"io"
	"os"

	"github.com/spf13/cobra"

	"github.com/geodro/lerd/internal/phpantom"
)

// NewLSPCmd returns the `lerd lsp` command: the PHP language server tinker uses,
// on stdio, for editors and AI agents that start one per project.
func NewLSPCmd() *cobra.Command {
	return &cobra.Command{
		Use:   "lsp",
		Short: "Run the PHP language server on stdio for editors and AI agents",
		Args:  cobra.NoArgs,
		RunE: func(cmd *cobra.Command, _ []string) error {
			dir, err := os.Getwd()
			if err != nil {
				return err
			}
			return runLSP(cmd.Context(), dir, os.Stdin, os.Stdout, os.Stderr)
		},
	}
}

// runLSP keeps stdout for the protocol alone, so a first-run download reports
// its progress on stderr.
func runLSP(ctx context.Context, dir string, in io.Reader, out, errOut io.Writer) error {
	if err := phpantom.EnsureBinary(ctx, errOut); err != nil {
		return err
	}
	cmd, err := phpantom.Command(ctx, dir)
	if err != nil {
		return err
	}
	cmd.Stdin, cmd.Stdout, cmd.Stderr = in, out, errOut
	return cmd.Run()
}
