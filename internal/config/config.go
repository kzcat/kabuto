package config

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"strconv"
	"strings"

	"github.com/kzcat/kabuto/internal/symbols"
)

type ItemConf struct {
	Name     string `json:"name"`
	Symbol   string `json:"symbol"`
	Country  string `json:"country"`
	Decimals int    `json:"decimals"`
}

type SectionConf struct {
	Key   string     `json:"key"`
	Title string     `json:"title"`
	Items []ItemConf `json:"items"`
}

type Config struct {
	Lang         string        `json:"lang"`
	Country      string        `json:"country"`
	Theme        string        `json:"theme"`
	Range        string        `json:"range"`
	Source       string        `json:"source"`
	Sections     []SectionConf `json:"sections"`
	SectionOrder []string      `json:"section_order"`
}

// DefaultPath returns the default config file path.
func DefaultPath() string {
	if dir := os.Getenv("XDG_CONFIG_HOME"); dir != "" {
		return filepath.Join(dir, "kabuto", "config.json")
	}
	home, err := os.UserHomeDir()
	if err != nil {
		return ""
	}
	return filepath.Join(home, ".config", "kabuto", "config.json")
}

// Load reads and parses config from path. Empty path or missing file returns zero Config (no error).
func Load(path string) (*Config, error) {
	if path == "" {
		return &Config{}, nil
	}
	data, err := os.ReadFile(path)
	if err != nil {
		if os.IsNotExist(err) {
			return &Config{}, nil
		}
		return nil, err
	}
	cfg, err := Parse(data)
	if err != nil {
		return nil, fmt.Errorf("kabuto: invalid config file %q: %v", path, err)
	}
	return cfg, nil
}

// Parse unmarshals JSON bytes into Config.
func Parse(data []byte) (*Config, error) {
	var cfg Config
	if err := json.Unmarshal(data, &cfg); err != nil {
		return nil, err
	}
	return &cfg, nil
}

// ParseAddSpec parses SYMBOL[:COUNTRY[:DECIMALS]].
func ParseAddSpec(spec string) (ItemConf, error) {
	parts := strings.Split(spec, ":")
	if len(parts) == 0 || parts[0] == "" {
		return ItemConf{}, fmt.Errorf("empty symbol in spec %q", spec)
	}
	ic := ItemConf{Symbol: parts[0], Decimals: 2}
	if len(parts) >= 2 {
		ic.Country = strings.ToUpper(parts[1])
	}
	if len(parts) >= 3 {
		d, err := strconv.Atoi(parts[2])
		if err != nil {
			return ItemConf{}, fmt.Errorf("invalid decimals in spec %q: %v", spec, err)
		}
		ic.Decimals = d
	}
	return ic, nil
}

// ToItem converts ItemConf to symbols.Item. If Name is empty, uses Symbol.
func (ic ItemConf) ToItem() symbols.Item {
	name := ic.Name
	if name == "" {
		name = ic.Symbol
	}
	return symbols.Item{Name: name, Symbol: ic.Symbol, Decimals: ic.Decimals, Country: ic.Country}
}

// RegisterSections registers config sections into symbols.Sections.
func RegisterSections(secs []SectionConf) {
	for _, sc := range secs {
		items := make([]symbols.Item, len(sc.Items))
		for i, ic := range sc.Items {
			items[i] = ic.ToItem()
		}
		title := sc.Title
		if title == "" {
			title = sc.Key
		}
		symbols.RegisterSection(symbols.Section{Key: sc.Key, Title: title, Items: items})
	}
}
