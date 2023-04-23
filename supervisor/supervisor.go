package supervisor

import (
	"errors"
	"github.com/oklog/ulid/v2"
	"math/rand"
	"os"
	"path/filepath"
	"strings"
	"time"
)

type Supervisor interface {
	Start() error
	Stop() error
	Setup() error
}

func GetOrCreateInstanceId(dir string) (ulid.ULID, error) {
	var ulidFileName = filepath.Join(dir, "ulid")
	var instanceId ulid.ULID
	if _, err := os.Stat(ulidFileName); err == nil {
		f, err := os.ReadFile(ulidFileName)
		if err != nil {
			panic(err)
		}
		rawUlid := strings.TrimSuffix(string(f), "\n")
		instanceId, err = ulid.Parse(rawUlid)
		return ulid.Parse(rawUlid)
	} else if errors.Is(err, os.ErrNotExist) {
		// Generate instance id.
		entropy := ulid.Monotonic(rand.New(rand.NewSource(0)), 0)
		instanceId, err := ulid.New(ulid.Timestamp(time.Now()), entropy)
		if err != nil {
			return instanceId, err
		}
		err = os.WriteFile(ulidFileName, []byte(instanceId.String()), 0644)
		return instanceId, err
	}
	return instanceId, nil
}
