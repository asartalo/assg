package config

import (
	"path/filepath"
	"strings"

	"github.com/BurntSushi/toml"
)

type ContentFeed struct {
	Name    string `toml:"name"`
	Title   string `toml:"title"`
	Include string `toml:"include"`
	Exclude string `toml:"exclude"`
}

var cfExclusions = make(map[string][]string)

func (cf ContentFeed) Exclusions() []string {
	key := strings.Trim(cf.Exclude, " ")
	ex, ok := cfExclusions[key]
	if !ok {
		splitted := strings.Split(cf.Exclude, ",")
		for _, str := range splitted {
			part := strings.Trim(str, " ")
			if part != "" {
				ex = append(ex, part)
			}
		}

		cfExclusions[key] = ex
	}

	return ex
}

var cfInclusions = make(map[string][]string)

func (cf ContentFeed) Inclusions() []string {
	key := strings.Trim(cf.Include, " ")
	in, ok := cfInclusions[key]
	if !ok {
		splitted := strings.Split(cf.Include, ",")
		for _, str := range splitted {
			part := strings.Trim(str, " ")
			if part != "" {
				in = append(in, part)
			}
		}

		cfInclusions[key] = in
	}

	return in
}

func (cf ContentFeed) IncludeAllInitially() bool {
	return cf.Include == ""
}

type Config struct {
	BaseURL          string           `toml:"base_url"`
	Title            string           `toml:"title"`
	Description      string           `toml:"description"`
	DefaultLanguage  string           `toml:"default_language"`
	Author           string           `toml:"author"`
	CompileSass      bool             `toml:"compile_sass"`
	GenerateFeed     bool             `toml:"generate_feed"`
	FeedLimit        int              `toml:"feed_limit"`
	FeedsForContent  []ContentFeed    `toml:"feeds_for_content"`
	Taxonomies       []TaxonomyConfig `toml:"taxonomies"`
	Markdown         MarkdownConfig   `toml:"markdown"`
	ContentDirectory string           `toml:"content_directory"`
	OutputDirectory  string           `toml:"output_directory"`
	IncludeDrafts    bool             `toml:"include_drafts"`
	Sitemap          bool             `toml:"sitemap"`
	PreBuildCmd      string           `toml:"prebuild"`
	PostBuildCmd     string           `toml:"postbuild"`
	ServerConfig     ServerConfig     `toml:"server"`
	DevMode          bool
	rootDirectory    string
}

type ServerConfig struct {
	Port        int64    `toml:"port"`
	WatchIgnore []string `toml:"watch_ignore"`
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

	if config.ServerConfig.Port == 0 {
		config.ServerConfig.Port = 8080
	}

	if len(config.FeedsForContent) == 0 {
		config.FeedsForContent = append(
			config.FeedsForContent,
			ContentFeed{Name: "atom"},
		)
	}
}
