package config

import (
	"encoding/json"
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"time"

	"github.com/dijkstra402/emoera-RAG-CLI/internal/apperr"
)

const (
	DefaultEndpoint = "https://rag.emoera.cn"
	DefaultProfile  = "default"
	DefaultOutput   = "table"
)

type Profile struct {
	Endpoint      string `json:"endpoint,omitempty"`
	DefaultOutput string `json:"defaultOutput,omitempty"`
}

type File struct {
	CurrentProfile string             `json:"currentProfile"`
	Profiles       map[string]Profile `json:"profiles"`
}

type Overrides struct {
	Profile  string
	Endpoint string
	Token    string
	Output   string
	Timeout  time.Duration
}

type Runtime struct {
	Profile     string
	Endpoint    string
	Token       string
	TokenSource string
	Output      string
	Timeout     time.Duration
	ConfigPath  string
}

func Path() (string, error) {
	if explicit := strings.TrimSpace(os.Getenv("EMOERA_CONFIG")); explicit != "" {
		return explicit, nil
	}
	dir, err := os.UserConfigDir()
	if err != nil {
		return "", apperr.Wrap(apperr.ExitConfiguration, "无法确定配置目录", err)
	}
	return filepath.Join(dir, "emoera", "config.json"), nil
}

func DefaultFile() File {
	return File{
		CurrentProfile: DefaultProfile,
		Profiles: map[string]Profile{
			DefaultProfile: {Endpoint: DefaultEndpoint, DefaultOutput: DefaultOutput},
		},
	}
}

func Load(path string) (File, error) {
	content, err := os.ReadFile(path)
	if errors.Is(err, os.ErrNotExist) {
		return DefaultFile(), nil
	}
	if err != nil {
		return File{}, apperr.Wrap(apperr.ExitConfiguration, "读取配置失败", err)
	}
	var file File
	if err := json.Unmarshal(content, &file); err != nil {
		return File{}, apperr.Wrap(apperr.ExitConfiguration, "配置文件不是有效 JSON", err)
	}
	normalize(&file)
	return file, nil
}

func Save(path string, file File) error {
	normalize(&file)
	if err := os.MkdirAll(filepath.Dir(path), 0o700); err != nil {
		return apperr.Wrap(apperr.ExitConfiguration, "创建配置目录失败", err)
	}
	content, err := json.MarshalIndent(file, "", "  ")
	if err != nil {
		return apperr.Wrap(apperr.ExitConfiguration, "生成配置失败", err)
	}
	content = append(content, '\n')
	if err := os.WriteFile(path, content, 0o600); err != nil {
		return apperr.Wrap(apperr.ExitConfiguration, "保存配置失败", err)
	}
	return nil
}

func Resolve(file File, path string, overrides Overrides, keychainToken string) (Runtime, error) {
	normalize(&file)
	profileName := firstNonBlank(overrides.Profile, os.Getenv("EMOERA_PROFILE"), file.CurrentProfile, DefaultProfile)
	profile, ok := file.Profiles[profileName]
	if !ok {
		return Runtime{}, apperr.New(apperr.ExitConfiguration, fmt.Sprintf("Profile %q 不存在", profileName))
	}

	endpoint := strings.TrimRight(firstNonBlank(overrides.Endpoint, os.Getenv("EMOERA_ENDPOINT"), profile.Endpoint, DefaultEndpoint), "/")
	output := strings.ToLower(firstNonBlank(overrides.Output, os.Getenv("EMOERA_OUTPUT"), profile.DefaultOutput, DefaultOutput))
	if output != "table" && output != "json" && output != "jsonl" {
		return Runtime{}, apperr.New(apperr.ExitConfiguration, "输出格式只能是 table、json 或 jsonl")
	}
	timeout := overrides.Timeout
	if timeout <= 0 {
		timeout = 30 * time.Second
		if raw := strings.TrimSpace(os.Getenv("EMOERA_TIMEOUT")); raw != "" {
			parsed, err := time.ParseDuration(raw)
			if err != nil || parsed <= 0 {
				return Runtime{}, apperr.New(apperr.ExitConfiguration, "EMOERA_TIMEOUT 不是有效时长")
			}
			timeout = parsed
		}
	}

	token := strings.TrimSpace(overrides.Token)
	tokenSource := "flag"
	if token == "" {
		token = strings.TrimSpace(os.Getenv("EMOERA_API_TOKEN"))
		tokenSource = "environment"
	}
	if token == "" {
		token = strings.TrimSpace(keychainToken)
		tokenSource = "keychain"
	}
	if token == "" {
		tokenSource = "none"
	}

	return Runtime{
		Profile: profileName, Endpoint: endpoint, Token: token,
		TokenSource: tokenSource, Output: output, Timeout: timeout, ConfigPath: path,
	}, nil
}

func Set(file *File, profileName, key, value string) error {
	normalize(file)
	profileName = firstNonBlank(profileName, file.CurrentProfile, DefaultProfile)
	profile := file.Profiles[profileName]
	switch strings.ToLower(strings.TrimSpace(key)) {
	case "endpoint":
		value = strings.TrimRight(strings.TrimSpace(value), "/")
		if value == "" {
			return apperr.New(apperr.ExitArguments, "endpoint 不能为空")
		}
		profile.Endpoint = value
	case "default-output", "defaultoutput", "output":
		value = strings.ToLower(strings.TrimSpace(value))
		if value != "table" && value != "json" && value != "jsonl" {
			return apperr.New(apperr.ExitArguments, "default-output 只能是 table、json 或 jsonl")
		}
		profile.DefaultOutput = value
	default:
		return apperr.New(apperr.ExitArguments, "仅支持 endpoint 和 default-output")
	}
	file.Profiles[profileName] = profile
	return nil
}

func Use(file *File, profileName string) error {
	normalize(file)
	profileName = strings.TrimSpace(profileName)
	if profileName == "" {
		return apperr.New(apperr.ExitArguments, "Profile 名称不能为空")
	}
	if _, ok := file.Profiles[profileName]; !ok {
		file.Profiles[profileName] = Profile{Endpoint: DefaultEndpoint, DefaultOutput: DefaultOutput}
	}
	file.CurrentProfile = profileName
	return nil
}

func normalize(file *File) {
	if file.Profiles == nil {
		file.Profiles = map[string]Profile{}
	}
	if strings.TrimSpace(file.CurrentProfile) == "" {
		file.CurrentProfile = DefaultProfile
	}
	if _, ok := file.Profiles[file.CurrentProfile]; !ok {
		file.Profiles[file.CurrentProfile] = Profile{Endpoint: DefaultEndpoint, DefaultOutput: DefaultOutput}
	}
}

func firstNonBlank(values ...string) string {
	for _, value := range values {
		if trimmed := strings.TrimSpace(value); trimmed != "" {
			return trimmed
		}
	}
	return ""
}
