package meta

import (
	"os"
)

func EnsureDirExists(dirName string) error {
	if _, err := os.Stat(dirName); os.IsNotExist(err) {
		err := os.Mkdir(dirName, 0755)
		if err != nil {
			return err
		}
	}
	return nil
}
