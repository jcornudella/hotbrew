package profile

// SourceSpec describes a source entry in a profile manifest.
type SourceSpec struct {
	Key        string   `yaml:"key"`
	Driver     string   `yaml:"driver"`
	Name       string   `yaml:"name"`
	Icon       string   `yaml:"icon"`
	ConfigKey  string   `yaml:"config_key,omitempty"`
	Queries    []string `yaml:"queries,omitempty"`
	Topics     []string `yaml:"topics,omitempty"`
	Tags       []string `yaml:"tags,omitempty"`
	Subreddits []string `yaml:"subreddits,omitempty"`
	Categories []string `yaml:"categories,omitempty"`
	FeedURL    string   `yaml:"feed_url,omitempty"`
}

// Profile groups the sources that should be registered.
type Profile struct {
	Sources []SourceSpec `yaml:"sources"`
}

// Default returns the built-in source profile used by the hosted newsletter.
func Default() *Profile {
	return &Profile{
		Sources: []SourceSpec{
			{
				Key:       "hackernews",
				Driver:    "hackernews",
				Name:      "Hacker News",
				Icon:      "🔶",
				ConfigKey: "hackernews",
			},
			{
				Key:     "hnsearch-claude",
				Driver:  "hnsearch",
				Name:    "Claude Code & Vibe Coding",
				Icon:    "🤖",
				Queries: []string{"Claude Code", "vibe coding", "AI coding assistant", "Anthropic Claude"},
			},
			{
				Key:    "github-trending",
				Driver: "github-trending",
				Name:   "GitHub Trending",
				Icon:   "🐙",
				Topics: []string{"ai", "llm", "machine-learning", "gpt", "claude"},
			},
			{
				Key:     "tldr-ai",
				Driver:  "tldr",
				Name:    "TLDR AI",
				Icon:    "🧠",
				FeedURL: "https://tldr.tech/api/rss/ai",
			},
			{
				Key:     "tldr-tech",
				Driver:  "tldr",
				Name:    "TLDR Tech",
				Icon:    "💻",
				FeedURL: "https://tldr.tech/api/rss/tech",
			},
			{
				Key:    "lobsters",
				Driver: "lobsters",
				Name:   "Lobste.rs",
				Icon:   "🦞",
				Tags:   []string{"ai", "ml", "programming", "compsci", "plt"},
			},
			{
				Key:        "reddit-ai",
				Driver:     "reddit",
				Name:       "Reddit AI",
				Icon:       "🔮",
				Subreddits: []string{"MachineLearning", "LocalLLaMA", "ClaudeAI"},
			},
			{
				Key:        "arxiv-llm",
				Driver:     "arxiv",
				Name:       "LLM Research",
				Icon:       "📄",
				Categories: []string{"cs.CL", "cs.AI", "cs.LG", "cs.MA"},
			},
		},
	}
}
