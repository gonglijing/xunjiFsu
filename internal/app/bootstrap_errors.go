package app

import "fmt"

func errDriverQuery(err error) error {
	if err == nil {
		return nil
	}
	return fmt.Errorf("failed to get drivers: %w", err)
}
