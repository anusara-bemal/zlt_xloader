package cmd

import (
	"encoding/json"
	"fmt"
	"os"
	"time"

	"github.com/spf13/cobra"

	"github.com/anusara-bemal/zlt-unlocker/internal/uart"
	"github.com/anusara-bemal/zlt-unlocker/pkg/safety"
)

var (
	uartPort     string
	uartBaud     int
	uartList     bool
	uartCommand  string
	uartScript   bool
	uartTimeout  time.Duration
	uartExpect   string
	uartSimulate bool
)

var uartCmd = &cobra.Command{
	Use:   "uart-at",
	Short: "Talk to the Tozed chipset over a serial/UART adapter",
	Long: `Opens a serial port and sends raw AT commands (e.g. AT^NVW,
AT+CLCK="PN",0) directly to the chipset.

WARNING: This requires a 3.3V TTL USB-UART adapter wired to the router's
serial header. Using RS232 voltages will destroy the device. The header
pinout is model-specific; consult hardware documentation before probing.

Use --simulate to run the same command engine against an in-memory mock
serial device, with no hardware required.`,
	RunE: func(cmd *cobra.Command, args []string) error {
		if uartList {
			ports, err := uart.ListPorts()
			if err != nil {
				return fmt.Errorf("list ports: %w", err)
			}
			fmt.Println("Available serial ports:")
			for _, p := range ports {
				fmt.Printf("  %s\n", p)
			}
			return nil
		}

		port := uartPort
		if port == "" {
			port = app.Profile.UART.Port
		}
		if port == "" && !uartSimulate {
			return fmt.Errorf("no serial port specified (use --port or set it in the profile)")
		}

		dryRun := app.DryRun && !uartSimulate
		if uartSimulate {
			fmt.Println("SIMULATE: using an in-memory mock serial device.")
		} else if !dryRun {
			if !safety.Confirm(fmt.Sprintf("Open %s and write AT commands to the chipset?", port), app.Yes) {
				return fmt.Errorf("aborted by operator")
			}
		} else {
			fmt.Printf("DRY-RUN: no port will be opened, no commands will be sent.\n\n")
		}

		var conn uart.Transport
		if uartSimulate {
			conn = uart.NewMock()
		} else if !dryRun {
			c, err := uart.Open(port, uartBaud,
				app.Profile.UART.DataBits, app.Profile.UART.StopBits,
				app.Profile.UART.Parity, uartTimeout)
			if err != nil {
				return err
			}
			defer c.Close()
			conn = c
		}

		engine := uart.NewEngine(conn, dryRun)

		if uartCommand != "" {
			resp, err := engine.Send(uartCommand, uartExpect, uartTimeout)
			if err != nil {
				return err
			}
			if app.JSON {
				return json.NewEncoder(os.Stdout).Encode(map[string]any{
					"command": uartCommand, "response": resp,
				})
			}
			fmt.Printf("> %s\n%s\n", uartCommand, resp)
			return nil
		}

		commands := app.Profile.UART.Commands
		if !uartScript {
			commands = nil
		}
		if len(commands) == 0 {
			return fmt.Errorf("no AT script configured; use --command or --script with a profile that defines commands")
		}

		results := engine.Run(commands, uartTimeout)
		if app.JSON {
			return json.NewEncoder(os.Stdout).Encode(results)
		}
		fmt.Printf("%-16s %-24s %s\n", "NAME", "COMMAND", "STATUS")
		for _, r := range results {
			status := "FAIL"
			if r.OK {
				status = "OK"
			}
			fmt.Printf("%-16s %-24s %s\n", r.Name, r.Command, status)
		}
		return nil
	},
}

func init() {
	uartCmd.Flags().StringVar(&uartPort, "port", "", "serial port (e.g. COM3, /dev/ttyUSB0)")
	uartCmd.Flags().IntVar(&uartBaud, "baud", 115200, "baud rate")
	uartCmd.Flags().BoolVar(&uartList, "list", false, "list available serial ports and exit")
	uartCmd.Flags().StringVar(&uartCommand, "command", "", "send a single raw AT command")
	uartCmd.Flags().BoolVar(&uartScript, "script", false, "run the AT script from the profile")
	uartCmd.Flags().DurationVar(&uartTimeout, "at-timeout", 2*time.Second, "per-command response timeout")
	uartCmd.Flags().StringVar(&uartExpect, "expect", "OK", "expected token for --command")
	uartCmd.Flags().BoolVar(&uartSimulate, "simulate", false, "run against an in-memory mock serial device")
	rootCmd.AddCommand(uartCmd)
}
