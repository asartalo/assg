package config

import (
	"github.com/BurntSushi/toml"
)

type Config struct {
	BaseURL         string                 `toml:"base_url"`
	Title           string                 `toml:"title"`
	Description     string                 `toml:"description"`
	DefaultLanguage string                 `toml:"default_language"`
	Author          string                 `toml:"author"`
	CompileSass     bool                   `toml:"compile_sass"`
	GenerateFeed    bool                   `toml:"generate_feed"`
	FeedLimit       int                    `toml:"feed_limit"`
	Taxonomies      []TaxonomyConfig       `toml:"taxonomies"`
	Markdown        MarkdownConfig         `toml:"markdown"`
	Extra           map[string]interface{} `toml:"extra"`
}

type TaxonomyConfig struct {
	Name       string `toml:"name"`
	Feed       bool   `toml:"feed"`
	PaginateBy int    `toml:"paginate_by"` // Add this field for optional pagination
}

type MarkdownConfig struct {
	HighlightCode    bool `toml:"highlight_code"`
	SmartPunctuation bool `toml:"smart_punctuation"`
}

func Load(filename string) (*Config, error) {
	var config Config
	_, err := toml.DecodeFile(filename, &config)
	if err != nil {
		return nil, err
	}
	return &config, nil
}
