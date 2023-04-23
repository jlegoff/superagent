package supervisor

import (
	"github.com/stretchr/testify/assert"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestMain(m *testing.M) {
	setup()
	code := m.Run()
	teardown()
	os.Exit(code)
}

func setup() {
	err := os.Mkdir("testdata/generated", 755)
	if err != nil {
		panic(err)
	}
}

func teardown() {
	err := os.RemoveAll("testdata/generated")
	if err != nil {
		panic(err)
	}
}

func TestGetUlid(t *testing.T) {
	instanceId, err := GetOrCreateInstanceId("testdata")
	assert.Nil(t, err)
	assert.Equal(t, instanceId.String(), "01GTEVKE9Q06AFVGQT5ZYC0GEK")
}

func TestCreateUlid(t *testing.T) {
	instanceId, err := GetOrCreateInstanceId("testdata/generated")
	assert.Nil(t, err)

	var ulidFileName = filepath.Join("testdata", "generated", "ulid")
	_, err = os.Stat(ulidFileName)
	assert.Nil(t, err)

	f, err := os.ReadFile(ulidFileName)
	assert.Nil(t, err)
	rawUlid := strings.TrimSuffix(string(f), "\n")
	assert.Equal(t, instanceId.String(), rawUlid)
}
