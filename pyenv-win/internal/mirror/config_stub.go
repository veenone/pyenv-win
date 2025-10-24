// +build !mirror

package mirror

import "fmt"

// MirrorConfig stub
type MirrorConfig struct{}

// LoadMirrorConfig stub
func LoadMirrorConfig(configPath string) (*MirrorConfig, error) {
	return nil, fmt.Errorf("mirror feature not compiled in this build")
}

// DefaultMirrorConfig stub
func DefaultMirrorConfig() *MirrorConfig {
	return &MirrorConfig{}
}
