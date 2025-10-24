// +build !mirror

package mirror

import (
	"fmt"

	"github.com/pyenv-win/pyenv-win/internal/config"
)

// MirrorPlugin is a stub when mirror feature is disabled
type MirrorPlugin struct{}

// NewMirrorPlugin returns an error when mirror feature is not compiled
func NewMirrorPlugin(cfg *config.PyenvConfig, configPath string) (*MirrorPlugin, error) {
	return nil, fmt.Errorf("mirror feature not compiled in this build\nRecompile with: go build -tags mirror")
}

// InitConfig is a stub
func InitConfig(path string) error {
	return fmt.Errorf("mirror feature not compiled in this build\nRecompile with: go build -tags mirror")
}

// MirrorAll is a stub
func (mp *MirrorPlugin) MirrorAll() error {
	return fmt.Errorf("mirror feature not available in this build")
}

// MirrorVersions is a stub
func (mp *MirrorPlugin) MirrorVersions(versions []string) error {
	return fmt.Errorf("mirror feature not available in this build")
}

// GenerateNexusMapping is a stub
func (mp *MirrorPlugin) GenerateNexusMapping() error {
	return fmt.Errorf("mirror feature not available in this build")
}
