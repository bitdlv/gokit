package random

import (
	"fmt"
	"hash/fnv"
	"os"
	"strconv"
	"sync"

	"github.com/bwmarrin/snowflake"
)

var (
	once       sync.Once
	node       *snowflake.Node
	initErr    error
	globalPref string
)

// Init initializes the Snowflake node using hashed hostname as machine ID
// and sets the default prefix for IDs
func Init(prefix string) error {
	once.Do(func() {
		globalPref = prefix
		machineID := generateMachineIDFromHostname()
		node, initErr = snowflake.NewNode(machineID)
	})
	return initErr
}

// GenerateID returns a unique ID as string (no prefix)
func GenerateID() string {
	if node == nil {
		panic("idgen: not initialized, call Init() first")
	}
	return node.Generate().String()
}

// GenerateIDWithPrefix returns a unique ID string with optional prefix
// If Init() was called with a prefix, it will be used automatically
func GenerateIDWithPrefix() string {
	if node == nil {
		panic("idgen: not initialized, call Init() first")
	}
	id := node.Generate().Int64()
	if globalPref != "" {
		return fmt.Sprintf("%s-%d", globalPref, id)
	}
	return strconv.FormatInt(id, 10)
}

// generateMachineIDFromHostname returns a hashed ID in range 0~1023
func generateMachineIDFromHostname() int64 {
	hostname, err := os.Hostname()
	if err != nil {
		// fallback default
		return 1
	}
	hasher := fnv.New32a()
	_, _ = hasher.Write([]byte(hostname))
	return int64(hasher.Sum32() % 1024)
}
