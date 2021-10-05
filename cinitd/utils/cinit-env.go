package utils

import (
	"os"
	"strings"
)

// CInitEnv for reading CINIT env into a map
type cInitEnv map[string]string

// GetCInitEnv will read environ and create a map of k:v from envs
// that have a CINIT_ prefix. The prefix is removed
func GetCInitEnv() map[string]string {
	var key string
	env := make(cInitEnv)
	osEnviron := os.Environ()
	cinitPrefix := "CINIT_"
	for _, b := range osEnviron {
		if strings.HasPrefix(b, cinitPrefix) {
			pair := strings.SplitN(b, "=", 2)
			key = strings.TrimPrefix(pair[0], cinitPrefix)
			key = strings.ToLower(key)
			key = strings.Replace(key, "_", ".", -1)
			env[key] = pair[1]
		}
	}

	return env
}
