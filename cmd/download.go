package cmd

import (
	"context"
	"fmt"
	"io"
	"net/http"
	"os"
	"path/filepath"
	"strings"
	"time"

	log "github.com/sirupsen/logrus"
	"github.com/spf13/cobra"

	"github.com/anusara-bemal/zlt-unlocker/pkg/netutil"
)

var (
	downloadRemote string
	downloadLocal  string
	downloadPort   int
)

var downloadCmd = &cobra.Command{
	Use:   "download",
	Short: "Export a file from the router via a local HTTP endpoint",
	Long: `Starts a local HTTP server, opens a root shell on the router and runs
curl there to upload the requested file. Used for configs, logs and NVRAM
dumps. The transfer runs over the shell, not the blocking HTTP injection.`,
	RunE: func(cmd *cobra.Command, args []string) error {
		if downloadRemote == "" {
			return fmt.Errorf("--remote is required")
		}
		if downloadLocal == "" {
			downloadLocal = filepath.Base(downloadRemote)
		}

		ip, err := netutil.LocalIP()
		if err != nil {
			return fmt.Errorf("local IP: %w", err)
		}

		received := make(chan int64, 1)
		mux := http.NewServeMux()
		mux.HandleFunc("/upload", func(w http.ResponseWriter, r *http.Request) {
			f, err := os.OpenFile(downloadLocal, os.O_CREATE|os.O_WRONLY|os.O_TRUNC, 0o600)
			if err != nil {
				http.Error(w, err.Error(), http.StatusInternalServerError)
				return
			}
			n, err := io.Copy(f, r.Body)
			_ = f.Close()
			if err != nil {
				http.Error(w, err.Error(), http.StatusInternalServerError)
				return
			}
			log.Infof("[download] received %d bytes", n)
			received <- n
			fmt.Fprint(w, "ok")
		})

		srv := &http.Server{
			Addr:              fmt.Sprintf(":%d", downloadPort),
			Handler:           mux,
			ReadHeaderTimeout: 5 * time.Second,
		}
		go func() {
			if err := srv.ListenAndServe(); err != nil && err != http.ErrServerClosed {
				log.WithError(err).Error("[download] server failed")
			}
		}()
		defer srv.Shutdown(context.Background())

		client, err := ensureShell()
		if err != nil {
			return err
		}
		defer client.Close()
		_, _ = client.Expect("# ", 3*time.Second)

		url := fmt.Sprintf("http://%s:%d/upload", ip, downloadPort)
		command := fmt.Sprintf("curl -s -T %s %s", downloadRemote, url)
		log.Infof("[download] transferring %s from router", downloadRemote)
		output, err := client.Run(command, "# ", 30*time.Second)
		if err != nil {
			return fmt.Errorf("curl on router: %w", err)
		}
		if strings.TrimSpace(output) != "" {
			log.Debugf("[download] router output: %q", output)
		}

		select {
		case n := <-received:
			log.Infof("[download] saved %d bytes to %s", n, downloadLocal)
			return nil
		case <-time.After(15 * time.Second):
			return fmt.Errorf("timed out waiting for upload from the router")
		}
	},
}

func init() {
	downloadCmd.Flags().StringVar(&downloadRemote, "remote", "", "remote path on the router")
	downloadCmd.Flags().StringVar(&downloadLocal, "local", "", "local output path")
	downloadCmd.Flags().IntVar(&downloadPort, "port", 8899, "local HTTP port")
	rootCmd.AddCommand(downloadCmd)
}
