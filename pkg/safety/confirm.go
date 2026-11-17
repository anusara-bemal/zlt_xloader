// Package safety contains interactive guards for destructive operations.
package safety

import (
	"bufio"
	"fmt"
	"os"
	"strings"
)

// Confirm asks the operator to confirm a destructive action.
// If yes is true (e.g. --yes flag), confirmation is skipped.
func Confirm(prompt string, yes bool) bool {
	if yes {
		return true
	}

	fmt.Printf("%s [y/N]: ", prompt)
	reader := bufio.NewReader(os.Stdin)
	answer, err := reader.ReadString('\n')
	if err != nil && answer == "" {
		return false
	}

	answer = strings.ToLower(strings.TrimSpace(answer))
	return answer == "y" || answer == "yes"
}
