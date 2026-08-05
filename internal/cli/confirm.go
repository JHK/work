package cli

import (
	"bufio"
	"fmt"
	"os"
	"strings"
)

// confirm asks on stdin; anything but y or yes cancels.
func confirm(what string) error {
	fmt.Printf("%s? [y/N] ", what)
	answer, err := bufio.NewReader(os.Stdin).ReadString('\n')
	if err != nil {
		fmt.Println()
		return errCancelled
	}
	if a := strings.ToLower(strings.TrimSpace(answer)); a == "y" || a == "yes" {
		return nil
	}
	return errCancelled
}
