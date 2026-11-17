package cmd

import (
	"encoding/hex"
	"fmt"
	"os"

	log "github.com/sirupsen/logrus"
	"github.com/spf13/cobra"

	cryptoutil "github.com/anusara-bemal/zlt-unlocker/internal/crypto"
	"github.com/anusara-bemal/zlt-unlocker/pkg/out"
)

var (
	cfgFile      string
	cfgAnalyze   bool
	cfgKey       string
	cfgAlgo      string
	cfgOut       string
	cfgHeader    int
	cfgSignature int
	cfgBlockSize int
	cfgSerial    string
	cfgIMEI      string
	cfgTop       int
)

var configDecryptCmd = &cobra.Command{
	Use:   "config-decrypt",
	Short: "Analyse and decrypt a Tozed S20 config.bin",
	Long: `Parses the config.bin container, reports per-block Shannon entropy,
and attempts heuristic key discovery using profile keys plus the device
serial/IMEI. With --key, that key is applied directly with the chosen algorithm.`,
	RunE: func(cmd *cobra.Command, args []string) error {
		if cfgFile == "" {
			return fmt.Errorf("--file is required")
		}

		container, err := cryptoutil.Parse(cfgFile, cfgHeader, cfgSignature, cfgBlockSize)
		if err != nil {
			return err
		}

		fmt.Println("=== Container ===")
		out.Table([][2]string{
			{"file", container.Path},
			{"size", fmt.Sprint(container.Size)},
			{"magic", valueOrUnknown(container.Magic)},
			{"header length", fmt.Sprint(container.HeaderLength)},
			{"payload offset", fmt.Sprint(container.PayloadOffset)},
			{"payload length", fmt.Sprint(container.PayloadLength)},
			{"signature length", fmt.Sprint(container.SignatureLength)},
		})

		payload, err := container.Payload(cfgFile)
		if err != nil {
			return err
		}
		entropy := cryptoutil.Shannon(payload)

		if cfgAnalyze {
			fmt.Println()
			fmt.Printf("Payload entropy: %.3f bits/byte  (%s)\n", entropy, cryptoutil.Classify(entropy))
			fmt.Println("Block report:")
			for i, b := range container.Blocks {
				if cfgTop > 0 && i >= cfgTop {
					fmt.Printf("  ... %d more blocks (raise --top to see them)\n", len(container.Blocks)-cfgTop)
					break
				}
				fmt.Printf("  %8d  %8d  entropy=%.3f  %s\n", b.Offset, b.Length, b.Entropy, b.Class)
			}
		}

		if cfgKey != "" {
			keyBytes := []byte(cfgKey)
			if decoded, err := hex.DecodeString(cfgKey); err == nil && len(decoded) > 0 {
				keyBytes = decoded
			}
			plain, err := cryptoutil.Decrypt(payload, keyBytes, cfgAlgo)
			if err != nil {
				return fmt.Errorf("decrypt: %w", err)
			}
			fmt.Println()
			fmt.Printf("Algorithm : %s\n", cfgAlgo)
			fmt.Printf("Score     : %.3f (lower is more plausible)\n", cryptoutil.Score(plain))
			fmt.Printf("Preview   : %s\n", cryptoutil.Preview(plain))

			if cfgOut != "" {
				if app.DryRun {
					log.Warnf("dry-run: would write %d bytes to %s", len(plain), cfgOut)
				} else if err := os.WriteFile(cfgOut, plain, 0o600); err != nil {
					return fmt.Errorf("write %q: %w", cfgOut, err)
				}
			}
			return nil
		}

		keys := cryptoutil.DeriveKeys(app.Profile.Crypto.KnownKeys, cfgSerial, cfgIMEI)
		algos := []string{"xor", "xor-add", "aes-cbc", "des-cbc"}
		log.Infof("trying %d keys x %d algorithms", len(keys), len(algos))
		candidates := cryptoutil.TryDecrypt(payload, keys, algos, cfgTop)

		fmt.Println()
		fmt.Println("=== Top candidates ===")
		for i, c := range candidates {
			fmt.Printf("%2d. score=%.3f  algo=%-8s key=%q\n    %s\n",
				i+1, c.Score, c.Algo, c.Key, c.Preview)
		}
		if len(candidates) == 0 {
			fmt.Println("No candidates produced parseable output; try --key with a known key.")
		}
		return nil
	},
}

func init() {
	configDecryptCmd.Flags().StringVar(&cfgFile, "file", "", "path to config.bin (required)")
	configDecryptCmd.Flags().BoolVar(&cfgAnalyze, "analyze", true, "print entropy / block analysis")
	configDecryptCmd.Flags().StringVar(&cfgKey, "key", "", "decrypt with this key (hex or ASCII)")
	configDecryptCmd.Flags().StringVar(&cfgAlgo, "algo", "xor", "algorithm: xor | xor-add | aes-cbc | des-cbc")
	configDecryptCmd.Flags().StringVar(&cfgOut, "out", "", "write decrypted bytes here (blocked by --dry-run)")
	configDecryptCmd.Flags().IntVar(&cfgHeader, "header", 0, "header length in bytes (0 = auto, -1 = none)")
	configDecryptCmd.Flags().IntVar(&cfgSignature, "signature", 0, "signature length in bytes (0 = auto, -1 = none)")
	configDecryptCmd.Flags().IntVar(&cfgBlockSize, "block-size", 4096, "entropy block size")
	configDecryptCmd.Flags().StringVar(&cfgSerial, "serial", "", "device serial number (key derivation)")
	configDecryptCmd.Flags().StringVar(&cfgIMEI, "imei", "", "device IMEI (key derivation)")
	configDecryptCmd.Flags().IntVar(&cfgTop, "top", 10, "maximum blocks/candidates to print")
	rootCmd.AddCommand(configDecryptCmd)
}
