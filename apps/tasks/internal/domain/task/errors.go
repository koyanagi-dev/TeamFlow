package task

import "fmt"

func ErrInvalidPatch(reason string) error {
	return fmt.Errorf("invalid patch: %s", reason)
}
