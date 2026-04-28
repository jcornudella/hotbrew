package clustering

import (
	"fmt"
	"os"
	"sort"
	"strings"
)

// buildSystemPrompt assembles the cacheable rubric that gets sent
// before every batch of clusters. The output is deterministic — slugs
// are sorted, examples are written inline as a constant — because any
// byte-level variance between calls would invalidate the prompt cache
// and turn what should be a 90%+ cache-hit workload into a 0% one.
//
// Target length is ≥ 4096 input tokens so that Opus 4.7 / Haiku 4.5
// will actually cache the prefix. Padding with worked examples not
// only clears the floor but improves classification quality on the
// tricky cases (DeepSeek not matching "ai" keywords, "robots" in
// tweets, markdown tooling not matching "devtools" keywords).
func buildSystemPrompt() string {
	var b strings.Builder
	b.WriteString(promptIntro)
	b.WriteString("\n\n## Theme catalog\n\n")
	b.WriteString(formatThemeCatalog())
	b.WriteString("\n\n## Few-shot examples\n\n")
	b.WriteString(promptExamples)
	b.WriteString("\n\n## Output format\n\n")
	b.WriteString(promptOutputInstructions)
	return b.String()
}

// formatThemeCatalog renders the canonical labels list in a stable
// order. Pulled from KnownLabels so adding a slug in labels.go
// automatically appears in the rubric — single source of truth.
func formatThemeCatalog() string {
	labels := KnownLabels()
	sort.Slice(labels, func(i, j int) bool { return labels[i].Slug < labels[j].Slug })

	var b strings.Builder
	for _, l := range labels {
		fmt.Fprintf(&b, "- **%s** (%s): %s\n", l.Slug, l.Display, themeGuidance[l.Slug])
		if len(l.keywords) > 0 {
			b.WriteString("  Indicative keywords: ")
			b.WriteString(strings.Join(l.keywords, ", "))
			b.WriteString("\n")
		}
	}
	return b.String()
}

// themeGuidance is a one-sentence prose description per slug. It
// complements the keyword list — descriptions cover semantic intent
// (so the model can tag DeepSeek as ai even though "deepseek" isn't
// a keyword), keywords are the literal lexical signals.
var themeGuidance = map[string]string{
	"ai":        "AI/ML research, LLM products, model releases, agents, prompt engineering, vendor news (Anthropic, OpenAI, DeepSeek, Mistral, Google AI, etc.). When a model or AI lab is named, this almost always wins.",
	"devtools":  "Developer tooling — IDEs, editors, CLIs, build systems, language runtimes, debuggers, terminal apps, version control, formatters, linters, frameworks for application code. Markdown editors, note-taking apps for developers, and text editors all count.",
	"infra":     "Infrastructure and platform — Kubernetes, databases (Postgres, MySQL, Redis, ClickHouse), message queues, cloud providers, edge/CDN, serverless, observability, networking, distributed systems internals.",
	"startups":  "Startup business news — funding rounds, acquisitions, IPOs, YC batches, hiring announcements at named companies, founder stories, valuations.",
	"papers":    "Research output — arXiv preprints, peer-reviewed studies, benchmarks, evaluations, formal results. Distinguishes from 'ai' when the focus is on the academic artifact itself.",
	"deep-read": "Long-form essays, postmortems, retrospectives, manifestos, deeply argued opinion pieces. Length and depth of argument matter more than topic.",
	"debate":    "Hot takes, controversial framings, X vs Y comparisons, critiques, reaction threads, polemics.",
	"repo":      "GitHub repositories — used as a section label for clusters whose items are all GitHub-trending. The matcher already handles this from source type; the LLM should rarely need to pick this directly.",
	"general":   "Catch-all for items that don't fit any of the above. Use sparingly — prefer the most specific applicable theme even when the match is approximate. Only fall back to general when no other theme is plausibly relevant.",
}

const promptIntro = `You are a content classifier for a personal news-digest application called hotbrew. The application clusters short articles, tweets, posts, and repository updates into themed sections. Your job is to assign each cluster a single theme slug from the fixed catalog below.

## Inputs

You receive a JSON array of clusters. Each cluster has:
- ` + "`id`" + `: an opaque identifier you must echo back unchanged.
- ` + "`items`" + `: a list of 1–10 short articles, each with ` + "`title`" + ` and ` + "`source`" + ` (e.g. "Hacker News", "Reddit AI", "X Bookmarks", "arxiv-llm").

## Decision principles

1. Pick the **most specific** theme that fits. A paper about an LLM is "ai" unless the cluster is dominated by paper-specific framing (benchmark numbers, methodology), in which case "papers".
2. **Source is a strong hint, not a verdict.** Reddit AI almost always means "ai", but a Reddit AI post about a Y Combinator funding round is "startups". X Bookmarks can be anything — read the title.
3. **One theme per cluster.** When a cluster looks like a 60/40 mix (e.g. an AI startup raise), pick the more specific one — "startups" beats "ai" when funding/acquisition is the focus, "ai" beats "startups" when the AI artifact is the focus.
4. **Avoid "general" unless nothing else fits.** Items about specific named technologies, products, or companies almost always have a more specific home.
5. **Use the keywords as floors, not ceilings.** If a tweet mentions "Claude" without using any AI-rubric keyword, it's still "ai". If a post mentions "neovim" but the actual content is about a YC startup, it's "startups".

The catalog below lists every legal slug. Returning a slug not in this catalog is a protocol error — the schema enforces this and the application will discard your output.`

// promptExamples is a hand-curated set of worked classifications.
// Two purposes: (1) bulk up the prompt to clear the 4096-token cache
// floor, (2) anchor the model on the trickier cases the keyword
// matcher gets wrong. Each example explains *why* — the rationale
// generalizes more than a bare input/output pair.
const promptExamples = `Below are worked examples covering common edge cases. Study these before classifying.

### Example 1 — DeepSeek (no AI keyword overlap)

Input cluster:
- "DeepSeek v4" (Hacker News)
- "Deepseek v4 people" (Reddit AI)

Theme: ` + "`ai`" + `
Why: "DeepSeek" is a major AI lab releasing a major model. The keyword matcher in labels.go has no entry for "deepseek", but the semantic intent is unambiguous — model releases from named AI labs always belong to ai.

### Example 2 — Tweet about robotics

Input cluster:
- "@pmarca Thanks! I think you'll like my Catechism for Robots which I just posted: https://t.co/rE7ZMMvIkE" (X Bookmarks)

Theme: ` + "`ai`" + `
Why: "Robots" in a tech-context tweet about a written piece is almost always about AI/ML embodiment, agents, or alignment — not industrial robotics. The X Bookmarks source is dominated by AI/tech content. When a tweet from a tech VC mentions robots, default to ai.

### Example 3 — Markdown tooling

Input cluster:
- "What are the best developer tools built around Markdown?" (Lobste.rs)

Theme: ` + "`devtools`" + `
Why: The keyword matcher has "developer tools" in the devtools rubric only via "ide", "editor", etc. — "developer tools" itself isn't a keyword. But the cluster is *literally about* developer tooling. Don't fall back to general just because the keyword match is weak.

### Example 4 — Funding round at an AI company

Input cluster:
- "Anthropic raises $5B at $200B valuation" (Hacker News)
- "Anthropic Series F" (Reddit AI)

Theme: ` + "`startups`" + `
Why: When the *story* is the funding event (raise, valuation, term sheet), the theme is startups even though the company is an AI company. The keyword "raises" wins over "anthropic" because it tells you what kind of news this is. If the same cluster were "Anthropic releases Claude 4.7", that's ai.

### Example 5 — Postgres announcement

Input cluster:
- "Postgres 18 released — async I/O and built-in OAuth" (Lobste.rs)
- "PostgreSQL 18 GA" (Hacker News)

Theme: ` + "`infra`" + `
Why: Database releases belong to infra. Multiple sources covering the same release strengthens the signal but doesn't change the slug.

### Example 6 — Long postmortem essay

Input cluster:
- "How we lost 11 hours of customer data — a postmortem" (Hacker News)

Theme: ` + "`deep-read`" + `
Why: Postmortems are the canonical deep-read shape. The story isn't about Kubernetes or Postgres specifically; it's a long-form retrospective on an outage. Even when the underlying tech is infra-flavored, the *form* of the post puts it in deep-read.

### Example 7 — arXiv preprint about LLM internals

Input cluster:
- "Different Language Models Learn Similar Number Representations" (arxiv-llm)

Theme: ` + "`papers`" + `
Why: An arXiv preprint with academic framing belongs to papers, not ai. The papers slug is for the academic artifact; ai is for products, releases, and applications. If unsure, source = arxiv is a near-certain signal for papers.

### Example 8 — GitHub trending repo about CLI tools

Input cluster:
- "charmbracelet/gum: A tool for glamorous shell scripts" (GitHub Trending)

Theme: ` + "`devtools`" + `
Why: Even though the source is github-trending (which has a special "repo" slug), the *content* fits devtools cleanly. The application's matcher will override this to "repo" when *every* item in the cluster is github-trending; if you see a mixed cluster, classify by content.

### Example 9 — Hot take / debate

Input cluster:
- "TypeScript is killing the dynamism of JavaScript and we're poorer for it" (Lobste.rs)
- "TypeScript users vs. JS users: a thread" (X Bookmarks)

Theme: ` + "`debate`" + `
Why: The first item is a polemic, the second is a "vs" thread — both signature debate shapes. The underlying topic is devtools, but the *form* (hot take, opposition framing) wins.

### Example 10 — Y Combinator hiring post

Input cluster:
- "SIM (YC X25) Is Hiring the Best Engineers in San Francisco" (Hacker News)

Theme: ` + "`startups`" + `
Why: YC + hiring + named company = startups, even when the hiring is for engineers at a tech company. The "startups" theme isn't only about funding — it covers the whole startup ecosystem (YC batches, founder stories, hiring rounds at named cos).

### Example 11 — Generic geopolitics or news

Input cluster:
- "US special forces soldier arrested after allegedly winning $400k on Maduro raid" (Hacker News)

Theme: ` + "`general`" + `
Why: This is news that ended up in the digest because Hacker News carried it, but it has no developer/AI/infra/startup angle. general is correct here. Use general only when *no* specific theme applies — not as a "I'm not sure" fallback.

### Example 12 — Vendor announcement about an AI product

Input cluster:
- "GPT-5.5 release 🚀, Anthropic $1T valuation 💰, DeepSeek v4" (Reddit AI)

Theme: ` + "`ai`" + `
Why: Even though "Anthropic $1T valuation" is a startups-flavored substring, the *cluster as a whole* is an AI roundup — three AI products/companies, framed as AI news. When a cluster aggregates AI news with one valuation tidbit, ai wins.

### Example 13 — LocalLLaMA rules update

Input cluster:
- "r/LocalLLaMa Rule Updates" (Reddit AI)

Theme: ` + "`ai`" + `
Why: Subreddit meta about an AI community is still ai. The source (Reddit AI) is a strong corroborating signal. Don't bounce this to general just because it's procedural — the *subject* is an AI community.

### Example 14 — Kubernetes operator paper

Input cluster:
- "Building a multi-tenant Kubernetes operator: lessons from production" (Lobste.rs)

Theme: ` + "`infra`" + `
Why: Kubernetes content goes to infra even when it's a long write-up. infra wins over deep-read here because the technical specificity (multi-tenant, operator pattern) matters more than the form. deep-read is reserved for content where the form *is* the story.

### Example 15 — Tweet linking a TLDR newsletter

Input cluster:
- "Today's TLDR: Cloudflare R2 outage, OpenAI Devs Day announcements, and a new Rust framework" (TLDR Tech)

Theme: ` + "`general`" + `
Why: TLDR digests are deliberately multi-topic — a single cluster of TLDR items can span infra, ai, and devtools. When the cluster mixes themes too thoroughly to pick one, general is correct (the digest itself will already split items into themed sections at the application layer). This is one of the few legitimate uses of general.

### Example 16 — Open-source LLM weights release

Input cluster:
- "Mistral releases 8x22B Mixtral with Apache 2.0 weights" (Reddit AI)
- "Mixtral 8x22B drops" (Hacker News)

Theme: ` + "`ai`" + `
Why: Even when the framing is "open-source release" (which sounds like devtools), an LLM weights release goes to ai. The artifact is a model, the audience is AI practitioners. Compare this to Example 5 (Postgres release): the theme follows the artifact type, not the release framing.

### Example 17 — Vector database product launch

Input cluster:
- "Pinecone launches serverless vector DB with 50% lower cost" (Hacker News)
- "Pinecone serverless announcement" (TLDR Tech)

Theme: ` + "`infra`" + `
Why: Vector databases are infrastructure, even when their primary use is AI applications. The decision rule: if the product is a database (Pinecone, Weaviate, pgvector) the theme is infra; if the product is a model or model-serving framework (Ollama, vLLM) the theme is ai. The "use case is AI" doesn't override the "category is database".

### Example 18 — Founder thread on hiring philosophy

Input cluster:
- "@founder Why we only hire engineers who can write a postmortem (a thread)" (X Bookmarks)

Theme: ` + "`debate`" + `
Why: A thread framed as opinionated philosophy with a contrarian framing ("only", "(a thread)") is debate, not deep-read. Threads are short-form by construction, so even when they're polemical they don't fit the deep-read pattern of long-form essays. If the same content were a 3,000-word blog post titled "Hiring engineers who can write a postmortem", it would be deep-read.

### Example 19 — Acquisition with no further detail

Input cluster:
- "Cloudflare acquires Magic Pocket" (Hacker News)

Theme: ` + "`startups`" + `
Why: Acquisitions are startups events even when the acquirer is a large public company. The lens here is "M&A activity in tech" rather than "Cloudflare news". Compare to "Cloudflare ships new R2 feature" which would be infra — same company, different theme based on what kind of news the cluster represents.

### Example 20 — RAG framework on GitHub Trending

Input cluster:
- "langchain-ai/langchain: Building applications with LLMs through composability" (GitHub Trending)
- "langchain hits 100K stars" (Hacker News)

Theme: ` + "`ai`" + `
Why: The cluster is mixed-source (one github-trending + one HN), so the auto-"repo" override doesn't apply (that requires *all* items to be github-trending). With mixed sources, classify by content: an LLM application framework is ai. Note: a pure github-trending cluster of the same repo would be auto-overridden to "repo" by the application after the LLM pass.

### Example 21 — eBPF / kernel internals article

Input cluster:
- "Tracing every system call with eBPF: a deep dive" (Lobste.rs)
- "eBPF tracing tutorial" (Hacker News)

Theme: ` + "`infra`" + `
Why: eBPF and kernel-level tracing belong to infra. The instinct might be "deep-read" because of the "deep dive" wording, but deep-read is reserved for postmortems, retrospectives, and manifestos — analytical content where the form *is* the story. A how-to/tutorial about a specific infra technology is infra even when long.

### Example 22 — AI safety / alignment essay

Input cluster:
- "Why current alignment techniques won't scale to superintelligence" (X Bookmarks)
- "Long-form: scalable oversight problems" (Reddit AI)

Theme: ` + "`ai`" + `
Why: AI safety and alignment content goes to ai, not deep-read or papers. The signal: when "alignment", "scalable oversight", or named alignment researchers (e.g. Christiano, Yudkowsky) are mentioned, the audience and framing are ai-centric. deep-read would be wrong because it dilutes the AI signal — the user follows ai precisely to see this kind of content.

### Example 23 — VS Code extension for Claude

Input cluster:
- "anthropic/claude-code-vscode: VS Code extension for Claude Code" (GitHub Trending)
- "Claude Code now in VS Code" (Hacker News)

Theme: ` + "`ai`" + `
Why: This is a mixed cluster (not all github-trending), and the artifact is an AI tool integration. The "VS Code extension" framing might pull toward devtools, but the substance — connecting an LLM to an editor — is squarely ai. Decision rule: when an integration brings AI capabilities into a non-AI surface, the theme is ai because the *novel* part is the AI, not the surface.

### Example 24 — Rust async runtime debate

Input cluster:
- "Tokio vs async-std: which async runtime should you pick in 2026?" (Lobste.rs)

Theme: ` + "`debate`" + `
Why: "X vs Y" comparisons are textbook debate even when the underlying topic is devtools. The cluster's theme is the *form* (a comparison/argument) rather than the *subject*. If the same article were titled "Tokio 1.40 release notes", that would be devtools.

### Example 25 — Stripe API breaking change

Input cluster:
- "Stripe announces v2026 API with breaking changes" (Hacker News)
- "How we're handling the Stripe v2026 migration" (Lobste.rs)

Theme: ` + "`devtools`" + `
Why: Third-party API changes that affect application code go to devtools. SDKs, public APIs, and integration tooling all sit in devtools rather than infra (which is reserved for the underlying platform components — databases, runtimes, message queues). Stripe is closer to a developer product than to infrastructure.

### Example 26 — AI agent framework launch

Input cluster:
- "Introducing Letta: open-source memory and agent framework" (Hacker News)
- "Letta GitHub repo trending" (Reddit AI)

Theme: ` + "`ai`" + `
Why: Agent frameworks (Letta, LangGraph, AutoGen, CrewAI) are ai, not devtools — even though they're "frameworks". The substance is LLM orchestration, agent loops, and memory systems, all squarely AI concepts. Devtools is for the substrate of writing application code (compilers, IDEs, build systems); ai is for the substrate of building AI applications.

### Example 27 — General security advisory

Input cluster:
- "CVE-2026-1234: critical RCE in libxml2" (Hacker News)
- "libxml2 patch released" (Lobste.rs)

Theme: ` + "`infra`" + `
Why: Security advisories for foundational libraries lean infra. The reasoning: libxml2 is plumbing — a library every server stack depends on. When the affected component is a runtime/library/protocol that systems run on, infra wins. If the CVE were in a developer-facing tool (a build system, an IDE), it would be devtools instead.`

const promptOutputInstructions = `Return JSON matching the schema below. Each input cluster gets one entry in ` + "`labels`" + `, with ` + "`id`" + ` echoed back unchanged. The order does not need to match the input order — the application will reorder by id.

Schema:

` + "```" + `json
{
  "labels": [
    {"id": "<cluster-id>", "slug": "<one of the catalog slugs>"}
  ]
}
` + "```" + `

Return only the JSON object — no prose, no markdown fences, no commentary. The application parses your output directly and any non-JSON content will produce a parse error.`

// getenv is os.Getenv wrapped so tests can stub the lookup. Production
// callers see the real environment; tests can swap this out.
var getenv = os.Getenv
