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
	{Name: "Comment", Pattern: `(?m)^\s*(?:#|//)[^\n]*`},
	{Name: "BlockComment", Pattern: `(?s)/\*.*?\*/`},
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
		participle.Elide("Comment", "BlockComment"),
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
			// Container header, ignore or handle if it has content
		case "Required":
			agent.Tools.Required = append(agent.Tools.Required, parseTools(content)...)
		case "Recommended":
			agent.Tools.Recommended = append(agent.Tools.Recommended, parseTools(content)...)
		}
	}

	return agent, nil
}

func parseTools(content string) []*MCPTools {
	var tools []*MCPTools
	lines := strings.Split(content, "\n")
	for _, line := range lines {
		name := strings.TrimSpace(line)
		if name != "" {
			tools = append(tools, &MCPTools{Name: name})
		}
	}
	return tools
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

// ParseFileForDisplay takes an io.Reader and tokenizes it for display.
func ParseFileForDisplay(r io.Reader) (*DisplayFile, error) {
	bytes, err := io.ReadAll(r)
	if err != nil {
		return nil, fmt.Errorf("reading failed: %w", err)
	}
	content := string(bytes)

	lex, err := agentLexer.Lex("", strings.NewReader(content))
	if err != nil {
		return nil, fmt.Errorf("lexing failed: %w", err)
	}

	tokens, err := lexer.ConsumeAll(lex)
	if err != nil {
		return nil, fmt.Errorf("consuming tokens failed: %w", err)
	}

	var displayTokens []Token
	for _, t := range tokens {
		val := t.Value
		switch t.Type {
		case agentLexer.Symbols()["Comment"]:
			displayTokens = append(displayTokens, Token{Comment: &val})
		case agentLexer.Symbols()["BlockComment"]:
			displayTokens = append(displayTokens, Token{BlockComment: &val})
		case agentLexer.Symbols()["Header"]:
			displayTokens = append(displayTokens, Token{Header: &val})
		case agentLexer.Symbols()["Line"]:
			displayTokens = append(displayTokens, Token{Line: &val})
		case agentLexer.Symbols()["Newline"]:
			displayTokens = append(displayTokens, Token{Newline: &val})
		}
	}

	return &DisplayFile{Tokens: displayTokens}, nil
}
