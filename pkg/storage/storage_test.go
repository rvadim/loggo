package storage

import (
	"os"
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestFileRegistry(t *testing.T) {
	registry, err := NewRegistryFile("/tmp/test.db", 1)
	assert.NoError(t, err)
	assert.Equal(t, "/tmp/test.db", registry.path)

	registry.Set("my-key.log", "123")
	output, err := registry.Get("my-key.log")
	assert.NoError(t, err)
	assert.Equal(t, "123", output)

	registry.Set("my-key.log", "321")
	output, err = registry.Get("my-key.log")
	assert.NoError(t, err)
	assert.Equal(t, "321", output)

	var all []string
	all, err = registry.GetAllKeys()
	assert.NoError(t, err)
	assert.Equal(t, "my-key.log", all[0])
	assert.Equal(t, 1, len(all))

	err = registry.Delete("my-key.log")
	assert.NoError(t, err)
	output, err = registry.Get("my-key.log")
	assert.NoError(t, err)
	assert.Equal(t, "", output)

	defer os.Remove("/tmp/test.db")
}
