package main

import (
	// "fmt"
	"gopkg.in/yaml.v3"
	"os"
	"path/filepath"
)

type AppConfig struct {
	Version    string       `yaml:"version"`
	Tools      []ToolConfig `yaml:"tools"`
	Env        EnvConfig    `yaml:"env,omitempty"`
	Categories []string     `yaml:"categories,omitempty"`
}

type ToolConfig struct {
	ID          string `yaml:"id"`
	Name        string `yaml:"name"`
	Type        string `yaml:"type"`
	Path        string `yaml:"path"`
	JavaVersion string `yaml:"javaVersion,omitempty"`
	Description string `yaml:"description,omitempty"`
	Category    string `yaml:"category"`
	// Priviledged bool   `yaml:"priviledged,omitempty"`
}

type EnvConfig struct {
	Java   []map[string]string `yaml:"java"`
	Python string              `yaml:"python"`
}

type ConfigManager struct {
	configPath string
}

func NewConfigManager() *ConfigManager {
	// appData, _ := os.UserConfigDir()
	// configPath := filepath.Join(appData, "ToolLauncher", "config.yaml")
	configPath := "config.yaml"
	return &ConfigManager{configPath: configPath}
}

func (cm *ConfigManager) LoadConfig() (*AppConfig, error) {
	data, err := os.ReadFile(cm.configPath)
	if err != nil {
		if os.IsNotExist(err) {
			// 返回默认空配置
			return &AppConfig{
				Version: "1.0",
				Tools:   []ToolConfig{},
			}, nil
		}
		return nil, err
	}

	var config AppConfig
	err = yaml.Unmarshal(data, &config)
	return &config, err
}

func (cm *ConfigManager) SaveConfig(config *AppConfig) error {
	// 确保目录存在
	os.MkdirAll(filepath.Dir(cm.configPath), 0755)

	data, err := yaml.Marshal(config)
	if err != nil {
		return err
	}

	return os.WriteFile(cm.configPath, data, 0644)
}

func (cm *ConfigManager) GetTools() ([]ToolConfig, error) {
	config, err := cm.LoadConfig()
	if err != nil {
		return nil, err
	}
	return config.Tools, nil
}

func (cm *ConfigManager) SaveTools(tools []ToolConfig) error {
	config, err := cm.LoadConfig()
	if err != nil {
		return err
	}

	config.Tools = tools
	return cm.SaveConfig(config)
}

func (cm *ConfigManager) GetEnvConfig() (*EnvConfig, error) {
	config, err := cm.LoadConfig()
	if err != nil {
		return nil, err
	}
	return &config.Env, nil
}

func (cm *ConfigManager) SaveEnvConfig(env *EnvConfig) error {
	config, err := cm.LoadConfig()
	if err != nil {
		return err
	}

	config.Env = *env
	return cm.SaveConfig(config)
}

// func (cm *ConfigManager) AddJavaVersion(version, path string) error {
// 	env, err := cm.GetEnvConfig()
// 	if err != nil {
// 		return err
// 	}

// 	// 检查版本是否已存在
// 	for _, javaMap := range env.Java {
// 		if javaMap[version] != "" {
// 			return fmt.Errorf("Java版本 %s 已存在", version)
// 		}
// 	}

// 	// 添加新版本
// 	newJavaMap := map[string]string{version: path}
// 	env.Java = append(env.Java, newJavaMap)

// 	return cm.SaveEnvConfig(env)
// }

// func (cm *ConfigManager) RemoveJavaVersion(version string) error {
// 	env, err := cm.GetEnvConfig()
// 	if err != nil {
// 		return err
// 	}

// 	// 查找并删除指定版本
// 	for i, javaMap := range env.Java {
// 		if javaMap[version] != "" {
// 			env.Java = append(env.Java[:i], env.Java[i+1:]...)
// 			return cm.SaveEnvConfig(env)
// 		}
// 	}

// 	return fmt.Errorf("未找到Java版本 %s", version)
// }

// func (cm *ConfigManager) UpdateJavaVersion(version, path string) error {
// 	env, err := cm.GetEnvConfig()
// 	if err != nil {
// 		return err
// 	}

// 	// 查找并更新指定版本
// 	for i, javaMap := range env.Java {
// 		if javaMap[version] != "" {
// 			env.Java[i] = map[string]string{version: path}
// 			return cm.SaveEnvConfig(env)
// 		}
// 	}

// 	// 如果没找到，添加新版本
// 	return cm.AddJavaVersion(version, path)
// }

func (cm *ConfigManager) GetCategories() ([]string, error) {
	config, err := cm.LoadConfig()
	if err != nil {
		return nil, err
	}
	if config.Categories == nil {
		return []string{}, nil
	}
	return config.Categories, nil
}

func (cm *ConfigManager) SaveCategories(categories []string) error {
	config, err := cm.LoadConfig()
	if err != nil {
		return err
	}
	config.Categories = categories
	return cm.SaveConfig(config)
}

// func (cm *ConfigManager) UpdatePythonPath(path string) error {
// 	env, err := cm.GetEnvConfig()
// 	if err != nil {
// 		return err
// 	}

// 	env.Python = path
// 	return cm.SaveEnvConfig(env)
// }
