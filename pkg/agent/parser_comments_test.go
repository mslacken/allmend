package agent

import (
	"strings"
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestParseAgentWithComments(t *testing.T) {
	agtContent := `%Meta
Name: TestAgent
# This is a comment
// This is another comment
/* Block comment
   should be ignored */
Description: An agent for testing comments.

%Manifest
This is the manifest.
# Comment inside manifest
// Another comment
/* Block comment inside manifest */
End of manifest.
`

	agent, err := ParseAgent(strings.NewReader(agtContent))
	assert.NoError(t, err)

	assert.Equal(t, "TestAgent", agent.Name)
	assert.Equal(t, "An agent for testing comments.", agent.Description)

	// Manifest content should ideally not contain comments if they are considered "fields" or structure.
	// However, usually Manifest content is raw text. The user request says "do not include fields which are commented".
	// If the parser treats everything as lines, comments in Manifest might be stripped too if we implement it at lexer level.
	// Let's assume for now that comments should be stripped everywhere.
	
	expectedManifest := `This is the manifest.
End of manifest.`
	// Normalize newlines
	gotManifest := strings.Join(strings.Fields(agent.Manifest.Content), " ")
	expectedManifestFields := strings.Join(strings.Fields(expectedManifest), " ")
	
	// Check if comments are present in Manifest content
	assert.NotContains(t, agent.Manifest.Content, "# Comment inside manifest")
	assert.NotContains(t, agent.Manifest.Content, "// Another comment")
	assert.NotContains(t, agent.Manifest.Content, "/* Block comment inside manifest */")
	
	assert.Equal(t, expectedManifestFields, gotManifest)
}
