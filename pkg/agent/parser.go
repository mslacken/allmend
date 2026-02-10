package agent

import (
	"fmt"
	"io"
	"strings"

	"github.com/alecthomas/participle/v2"
	"github.com/alecthomas/participle/v2/lexer"
)

// Define the lexer to handle Section Headers and raw text lines.
// We treat the file as a sequence of headers followed by lines of text.
var agentLexer = lexer.MustSimple([]lexer.SimpleRule{
	{Name: "Header", Pattern: `(?m)^%[a-zA-Z]+`},
	{Name: "Line", Pattern: `(?m)^[^%\r\n].*$`}, // Matches any line that doesn't start with '%'
	{Name: "Newline", Pattern: `\r?\n`},
})

// AST Structures
type AgentFile struct {
	Sections []*Section `parser:"@@*"`
}

type Section struct {
	Header string   `parser:"@Header"`
	Lines  []string `parser:"(@Line | @Newline)*"`
}

// ParseAgent parses a flat .agt file into an Agent struct using participle.
func ParseAgent(r io.Reader) (*Agent, error) {
	// Create parser with custom lexer
	// We do not elide Newline because we need it to preserve text structure in Manifests.
	parser := participle.MustBuild[AgentFile](
		participle.Lexer(agentLexer),
	)

	// Parse
	ast, err := parser.Parse("", r)
	if err != nil {
		return nil, fmt.Errorf("parsing failed: %w", err)
	}

	// Map AST to Agent struct
	agent := &Agent{
		Manifest: &AgentManifest{},
		Mission:  &AgentMission{},
		Tools:    &AgentTools{},
		Meta:     &AgentMeta{},
	}

	for _, sec := range ast.Sections {
		headerName := strings.TrimPrefix(strings.TrimSpace(sec.Header), "%")

		var contentBuilder strings.Builder
		for _, l := range sec.Lines {
			contentBuilder.WriteString(l)
		}

		content := strings.TrimSpace(contentBuilder.String())

		switch headerName {
		case "Meta":
			parseMetaLines(content, agent)
		case "Manifest":
			agent.Manifest.Content = content
		case "Mission":
			agent.Mission.Content = content
		case "Description":
			agent.Description = content
		case "Tools":
			// Placeholder
		}
	}

	return agent, nil
}

func parseMetaLines(content string, agent *Agent) {
	lines := strings.Split(content, "\n")
	for _, line := range lines {
		parts := strings.SplitN(line, ":", 2)
		if len(parts) != 2 {
			continue
		}
		key := strings.TrimSpace(strings.ToLower(parts[0]))
		val := strings.TrimSpace(parts[1])

		switch key {
		case "name":
			agent.Name = val
		case "description":
			agent.Description = val
		case "author":
			agent.Meta.Author = val
		case "version":
			agent.Meta.Version = val
		}
	}
}
