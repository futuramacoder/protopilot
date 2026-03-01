package main

import (
	"fmt"
	"os"

	tea "charm.land/bubbletea/v2"
	"github.com/spf13/cobra"

	"github.com/futuramacoder/protopilot/internal/app"
	grpcpkg "github.com/futuramacoder/protopilot/internal/grpc"
)

func main() {
	var (
		protoPaths  []string
		host        string
		plaintext   bool
		importPaths []string
		caCert      string
		cert        string
		key         string
		serverName  string
	)

	rootCmd := &cobra.Command{
		Use:   "protopilot",
		Short: "Interactive TUI client for gRPC services",
		Long:  "Protopilot reads .proto files at runtime and provides an interactive TUI for exploring and calling gRPC services.",
		RunE: func(cmd *cobra.Command, args []string) error {
			if len(protoPaths) == 0 {
				return fmt.Errorf("at least one --proto file is required")
			}

			cfg := app.Config{
				ProtoPaths:  protoPaths,
				Host:        host,
				ImportPaths: importPaths,
				TLS: grpcpkg.TLSConfig{
					Plaintext:  plaintext,
					CACert:     caCert,
					Cert:       cert,
					Key:        key,
					ServerName: serverName,
				},
			}

			model := app.New(cfg)
			p := tea.NewProgram(model)
			_, err := p.Run()
			return err
		},
	}

	flags := rootCmd.Flags()
	flags.StringArrayVarP(&protoPaths, "proto", "p", nil, "Proto file paths (required, repeatable)")
	flags.StringVar(&host, "host", "localhost:50051", "gRPC server host:port")
	flags.BoolVar(&plaintext, "plaintext", false, "Disable TLS (use insecure connection)")
	flags.StringArrayVar(&importPaths, "import-path", nil, "Additional proto import paths (repeatable)")
	flags.StringVar(&caCert, "cacert", "", "CA certificate file for TLS")
	flags.StringVar(&cert, "cert", "", "Client certificate file for mTLS")
	flags.StringVar(&key, "key", "", "Client private key file for mTLS")
	flags.StringVar(&serverName, "servername", "", "TLS server name override (SNI)")

	if err := rootCmd.Execute(); err != nil {
		os.Exit(1)
	}
}
