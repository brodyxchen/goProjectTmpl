package config

import (
	"encoding/json"
	"errors"
	"fmt"
	"strings"
	"sync"

	"github.com/BurntSushi/toml"
	"github.com/spf13/cast"
	"gopkg.in/yaml.v3"
)

var (
	// ErrConfigNotExist 配置不存在
	ErrConfigNotExist = errors.New("app/config: config not exist")
	// ErrProviderNotExist provider不存在
	ErrProviderNotExist = errors.New("app/config: provider not exist")
	// ErrCodecNotExist codec不存在
	ErrCodecNotExist = errors.New("app/config: codec not exist")
)

func init() {
	RegisterCodec(&YamlCodec{})
	RegisterCodec(&JSONCodec{})
	RegisterCodec(&TomlCodec{})
}

// LoadOption 配置加载选项
type LoadOption func(*FrameworkConfig)

// FullConfigLoader 创建一个Config实例
type FullConfigLoader struct {
	configMap map[string]Config
	rwl       sync.RWMutex
}

// Load 根据参数加载指定配置
func (loader *FullConfigLoader) Load(path string, opts ...LoadOption) (Config, error) {
	yc := newFullConfig(path)
	for _, o := range opts {
		o(yc)
	}

	if yc.decoder == nil {
		return nil, ErrCodecNotExist
	}

	if yc.p == nil {
		return nil, ErrProviderNotExist
	}

	key := fmt.Sprintf("%s.%s.%s", yc.decoder.Name(), yc.p.Name(), path)
	loader.rwl.RLock()
	if c, ok := loader.configMap[key]; ok {
		loader.rwl.RUnlock()
		return c, nil
	}
	loader.rwl.RUnlock()

	err := yc.Load()
	if err != nil {
		return nil, err
	}

	loader.rwl.Lock()
	loader.configMap[key] = yc
	loader.rwl.Unlock()

	yc.p.Watch(func(p string, data []byte) {
		if p == path {
			loader.rwl.Lock()
			delete(loader.configMap, key)
			loader.rwl.Unlock()
		}
	})

	return yc, nil
}

// Reload 重新加载
func (loader *FullConfigLoader) Reload(path string, opts ...LoadOption) error {
	yc := newFullConfig(path)
	for _, o := range opts {
		o(yc)
	}
	key := fmt.Sprintf("%s.%s.%s", yc.decoder.Name(), yc.p.Name(), path)
	loader.rwl.RLock()
	if config, ok := loader.configMap[key]; ok {
		loader.rwl.RUnlock()
		config.Reload()
		return nil
	}
	loader.rwl.RUnlock()
	return ErrConfigNotExist
}

func newFullConfigLoad() *FullConfigLoader {
	return &FullConfigLoader{configMap: map[string]Config{}, rwl: sync.RWMutex{}}
}

// DefaultConfigLoader 默认配置加载器
var DefaultConfigLoader = newFullConfigLoad()

// YamlCodec 解码Yaml
type YamlCodec struct{}

// Name yaml codec
func (*YamlCodec) Name() string {
	return "yaml"
}

// Unmarshal yaml decode
func (c *YamlCodec) Unmarshal(in []byte, out interface{}) error {
	return yaml.Unmarshal(in, out)
}

// JSONCodec JSON codec
type JSONCodec struct{}

// Name JSON codec
func (*JSONCodec) Name() string {
	return "json"
}

// Unmarshal JSON decode
func (c *JSONCodec) Unmarshal(in []byte, out interface{}) error {
	return json.Unmarshal(in, out)
}

// TomlCodec toml codec
type TomlCodec struct{}

// Name toml codec
func (*TomlCodec) Name() string {
	return "toml"
}

// Unmarshal toml decode
func (c *TomlCodec) Unmarshal(in []byte, out interface{}) error {
	return toml.Unmarshal(in, out)
}

// FrameworkConfig 解析yaml类型的配置文件
type FrameworkConfig struct {
	p             DataProvider
	unmarshedData interface{}
	path          string
	decoder       Codec
	rawData       []byte
}

func (c *FrameworkConfig) find(key string) (interface{}, error) {
	subkeys := c.parseKey(key)
	return c.locateSubkey(subkeys)
}

// Get 根据key读取配置
func (c *FrameworkConfig) Get(key string, defaultValue interface{}) interface{} {
	if v, err := c.find(key); err == nil {
		return v
	}

	return defaultValue
}

// Bytes 获得原始配置
func (c *FrameworkConfig) Bytes() []byte {
	return c.rawData
}

func (c *FrameworkConfig) findWithDefaultValue(key string, defaultValue interface{}) interface{} {
	v, err := c.find(key)
	if err != nil {
		return defaultValue
	}

	switch defaultValue.(type) {
	case bool:
		v, err = cast.ToBoolE(v)
	case string:
		v, err = cast.ToStringE(v)
	case int:
		v, err = cast.ToIntE(v)
	case int32:
		v, err = cast.ToInt32E(v)
	case int64:
		v, err = cast.ToInt64E(v)
	case uint:
		v, err = cast.ToUintE(v)
	case uint32:
		v, err = cast.ToUint32E(v)
	case uint64:
		v, err = cast.ToUint64E(v)
	case float64:
		v, err = cast.ToFloat64E(v)
	case float32:
		v, err = cast.ToFloat32E(v)
	default:
	}

	if err != nil {
		return defaultValue
	}

	return v
}

// GetInt 根据key读取int类型配置
func (c *FrameworkConfig) GetInt(key string, defaultValue int) int {
	return cast.ToInt(c.findWithDefaultValue(key, defaultValue))
}

// GetInt32 根据key读取int32类型配置
func (c *FrameworkConfig) GetInt32(key string, defaultValue int32) int32 {
	return cast.ToInt32(c.findWithDefaultValue(key, defaultValue))
}

// GetInt64 根据key读取int64类型配置
func (c *FrameworkConfig) GetInt64(key string, defaultValue int64) int64 {
	return cast.ToInt64(c.findWithDefaultValue(key, defaultValue))
}

// GetUint 根据key读取int类型配置
func (c *FrameworkConfig) GetUint(key string, defaultValue uint) uint {
	return cast.ToUint(c.findWithDefaultValue(key, defaultValue))
}

// GetUint32 根据key读取uint32类型配置
func (c *FrameworkConfig) GetUint32(key string, defaultValue uint32) uint32 {
	return cast.ToUint32(c.findWithDefaultValue(key, defaultValue))
}

// GetUint64 根据key读取uint64类型配置
func (c *FrameworkConfig) GetUint64(key string, defaultValue uint64) uint64 {
	return cast.ToUint64(c.findWithDefaultValue(key, defaultValue))
}

// GetFloat64 根据key读取float64类型配置
func (c *FrameworkConfig) GetFloat64(key string, defaultValue float64) float64 {
	return cast.ToFloat64(c.findWithDefaultValue(key, defaultValue))
}

// GetFloat32 根据key读取float32类型配置
func (c *FrameworkConfig) GetFloat32(key string, defaultValue float32) float32 {
	return cast.ToFloat32(c.findWithDefaultValue(key, defaultValue))
}

// GetBool 根据key读取bool类型配置
func (c *FrameworkConfig) GetBool(key string, defaultValue bool) bool {
	return cast.ToBool(c.findWithDefaultValue(key, defaultValue))
}

// IsSet 根据key判断配置是否存在
func (c *FrameworkConfig) IsSet(key string) bool {
	subkeys := c.parseKey(key)
	_, err := c.locateSubkey(subkeys)
	if err != nil {
		return false
	}
	return true
}

func (c *FrameworkConfig) locateSubkey(subkeys []string) (interface{}, error) {
	return c.search(cast.ToStringMap(c.unmarshedData), subkeys)
}

func (c *FrameworkConfig) search(unmarshedData map[string]interface{}, subkeys []string) (interface{}, error) {
	if len(subkeys) == 0 {
		return nil, ErrConfigNotExist
	}

	next, ok := unmarshedData[subkeys[0]]
	if ok {
		if len(subkeys) == 1 {
			return next, nil
		}

		switch next.(type) {

		case map[interface{}]interface{}:
			return c.search(cast.ToStringMap(next), subkeys[1:])

		case map[string]interface{}:
			return c.search(next.(map[string]interface{}), subkeys[1:])

		default:
			return nil, ErrConfigNotExist
		}
	}
	return nil, ErrConfigNotExist
}

// GetString 根据key读取string类型配置
func (c *FrameworkConfig) GetString(key string, defaultValue string) string {
	subkeys := c.parseKey(key)

	value, err := c.locateSubkey(subkeys)
	if err != nil {
		return defaultValue
	}

	if result, ok := value.(string); ok {
		return result
	}

	if result, err := cast.ToStringE(value); err == nil {
		return result
	}

	return defaultValue
}

// Load 加载配置
func (c *FrameworkConfig) Load() error {
	if c.p == nil {
		return ErrProviderNotExist
	}

	data, err := c.p.Read(c.path)
	if err != nil {
		return fmt.Errorf("app/config: failed to load %s: %s", c.path, err.Error())
	}
	c.rawData = data
	c.unmarshedData = map[string]interface{}{}
	err = c.decoder.Unmarshal(c.rawData, &c.unmarshedData)
	if err != nil {
		return fmt.Errorf("app/config: failed to parse %s: %s", c.path, err.Error())
	}
	return nil
}

// Reload 重新载入
func (c *FrameworkConfig) Reload() {
	if c.p == nil {
		return
	}

	data, err := c.p.Read(c.path)
	if err != nil {
		fmt.Printf("app/config: failed to reload %s: %v", c.path, err)
		return
	}

	unmarshedData := map[string]interface{}{}
	if err = c.decoder.Unmarshal(data, &unmarshedData); err != nil {
		fmt.Printf("app/config: failed to parse %s: %v", c.path, err)
		return
	}

	c.rawData = data
	c.unmarshedData = unmarshedData
}

// Unmarshal 反序列化
func (c *FrameworkConfig) Unmarshal(out interface{}) error {
	return c.decoder.Unmarshal(c.rawData, out)
}

func (c *FrameworkConfig) parseKey(key string) []string {
	return strings.Split(key, ".")
}

func newFullConfig(path string) *FrameworkConfig {
	yc := &FrameworkConfig{
		p:       GetProvider("file"),
		path:    path,
		decoder: &YamlCodec{},
	}
	return yc
}
