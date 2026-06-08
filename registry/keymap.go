package registry

// KeymapRegistry maps (mode, key) → command name.
// V1 is single-key only. Multi-key sequences (e.g. "gg") are future work.
type KeymapRegistry struct {
	bindings map[string]map[int32]string // mode → key → command
}

func NewKeymapRegistry() *KeymapRegistry {
	return &KeymapRegistry{
		bindings: make(map[string]map[int32]string),
	}
}

// Register binds a key to a command in a given mode.
func (k *KeymapRegistry) Register(mode string, key int32, command string) {
	if k.bindings[mode] == nil {
		k.bindings[mode] = make(map[int32]string)
	}
	k.bindings[mode][key] = command
}

// Resolve looks up the command for a key in the given mode.
func (k *KeymapRegistry) Resolve(mode string, key int32) (string, bool) {
	cmds, ok := k.bindings[mode]
	if !ok {
		return "", false
	}
	cmd, ok := cmds[key]
	return cmd, ok
}
