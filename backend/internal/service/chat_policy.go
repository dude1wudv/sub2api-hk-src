package service

import "maps"

// Policy values contain only scalar fields; a map clone fully isolates snapshots.
func CloneChatModelPolicies(value map[string]ChatModelPolicy) map[string]ChatModelPolicy {
	return maps.Clone(value)
}
