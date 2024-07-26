package config

import (
	"path/filepath"

	"github.com/BurntSushi/toml"
)

type Config struct {
	BaseURL          string                 `toml:"base_url"`
	Title            string                 `toml:"title"`
	Description      string                 `toml:"description"`
	DefaultLanguage  string                 `toml:"default_language"`
	Author           string                 `toml:"author"`
	CompileSass      bool                   `toml:"compile_sass"`
	GenerateFeed     bool                   `toml:"generate_feed"`
	FeedLimit        int                    `toml:"feed_limit"`
	Taxonomies       []TaxonomyConfig       `toml:"taxonomies"`
	Markdown         MarkdownConfig         `toml:"markdown"`
	Extra            map[string]interface{} `toml:"extra"`
	ContentDirectory string                 `toml:"content_directory"`
	OutputDirectory  string                 `toml:"output_directory"`
	IncludeDrafts    bool                   `toml:"include_drafts"`
	rootDirectory    string
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

func (c *Config) RootDirectory() string {
	return c.rootDirectory
}

func (c *Config) ContentDirectoryAbsolute() string {
	if filepath.IsAbs(c.ContentDirectory) {
		return c.ContentDirectory
	}

	return filepath.Join(c.rootDirectory, c.ContentDirectory)
}

func (c *Config) OutputDirectoryAbsolute() string {
	if filepath.IsAbs(c.OutputDirectory) {
		return c.OutputDirectory
	}

	return filepath.Join(c.rootDirectory, c.OutputDirectory)
}

func Load(filename string) (*Config, error) {
	var config Config
	_, err := toml.DecodeFile(filename, &config)
	if err != nil {
		return nil, err
	}

	config.rootDirectory = filepath.Dir(filename)
	setDefaults(&config)

	return &config, nil
}

func setDefaults(config *Config) {
	if config.ContentDirectory == "" {
		config.ContentDirectory = "content"
	}

	if config.OutputDirectory == "" {
		config.OutputDirectory = "public"
	}
}
