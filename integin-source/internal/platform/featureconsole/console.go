package featureconsole

import (
	"errors"
	"strings"
	"sync"
)

type Module struct {
	Name          string
	Version       string
	EntryPoint    string
	Subscriptions []string
	Emissions     []string
	ConfigSchema  map[string]string
	Environment   string
}
type Update struct {
	ModuleName  string
	FromVersion string
	ToVersion   string
	Path        []string
}
type Console struct {
	mu      sync.RWMutex
	modules map[string]Module
}

func New() *Console { return &Console{modules: make(map[string]Module)} }
func (c *Console) Register(module Module) error {
	if strings.TrimSpace(module.Name) == "" || strings.TrimSpace(module.Version) == "" || strings.TrimSpace(module.EntryPoint) == "" {
		return errors.New("module name, version, and entry point are required")
	}
	module.Subscriptions = append([]string(nil), module.Subscriptions...)
	module.Emissions = append([]string(nil), module.Emissions...)
	module.ConfigSchema = cloneSchema(module.ConfigSchema)
	c.mu.Lock()
	defer c.mu.Unlock()
	c.modules[module.Name] = module
	return nil
}
func (c *Console) Installed() []Module {
	c.mu.RLock()
	defer c.mu.RUnlock()
	result := make([]Module, 0, len(c.modules))
	for _, module := range c.modules {
		module.Subscriptions = append([]string(nil), module.Subscriptions...)
		module.Emissions = append([]string(nil), module.Emissions...)
		module.ConfigSchema = cloneSchema(module.ConfigSchema)
		result = append(result, module)
	}
	return result
}
func (c *Console) UpdatePath(moduleName, fromVersion, toVersion string) (Update, error) {
	if moduleName == "" || fromVersion == "" || toVersion == "" {
		return Update{}, errors.New("module and versions are required")
	}
	if fromVersion == toVersion {
		return Update{}, errors.New("versions must differ")
	}
	c.mu.RLock()
	_, ok := c.modules[moduleName]
	c.mu.RUnlock()
	if !ok {
		return Update{}, errors.New("module not installed")
	}
	return Update{ModuleName: moduleName, FromVersion: fromVersion, ToVersion: toVersion, Path: []string{"TESTING", "PROMOTION", "LIVE", "HEALTH_CHECK"}}, nil
}
func cloneSchema(schema map[string]string) map[string]string {
	cloned := make(map[string]string, len(schema))
	for key, value := range schema {
		cloned[key] = value
	}
	return cloned
}
