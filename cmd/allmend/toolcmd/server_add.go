package toolcmd

import (
	"context"
	"fmt"
	"net/url"
	"os"
	"text/tabwriter"

	"github.com/SUSE/allmend/pkg/mcp"
	"github.com/SUSE/allmend/pkg/tool"
	"github.com/spf13/cobra"
)

var (
	serverType    string
	serverCommand string
	serverArgs    []string
	serverEnv     []string
)

var serverAddCmd = &cobra.Command{
	Use:   "serveradd [NAME] [URL or COMMAND]",
	Short: "Add a new tool server via MCP",
	Long: `Add a new tool server.
	
For HTTP servers:
  allmend tool serveradd my-server http://localhost:8080

For Stdio servers:
  allmend tool serveradd my-local-server --type stdio --command "npx" --args "-y,@modelcontextprotocol/server-memory"
  
  Note: For args, use comma separated values or multiple --args flags.`,
	Args: cobra.MinimumNArgs(1),
	Run: func(cmd *cobra.Command, args []string) {
		ctx := context.Background()
		var name, connectionStr string
		
		if len(args) >= 2 {
			name = args[0]
			connectionStr = args[1]
		} else if len(args) == 1 {
			if serverType == "stdio" && serverCommand != "" {
				// If --command is provided, the positional arg is the name
				name = args[0]
				connectionStr = serverCommand
			} else {
				// Otherwise positional arg is the connection string (URL or command)
				connectionStr = args[0]
				if serverType == "http" {
					name = connectionStr
				} else {
					// For stdio without --command, use command as name
					name = connectionStr
				}
			}
		}

		if name == "" && serverType == "stdio" && serverCommand != "" {
			fmt.Println("Error: Please provide a name for the server as a positional argument.")
			return
		}

		// Configure Transport
		var transport mcp.Transport
		var server tool.Server

		if serverType == "http" {
			// Validate URL
			if _, err := url.ParseRequestURI(connectionStr); err != nil {
				fmt.Printf("Error: Invalid URL provided: %v\n", err)
				return
			}
			transport = mcp.NewHTTPTransport(connectionStr)
			server = tool.Server{
				Name: name,
				Type: "http",
				URL:  connectionStr,
			}
		} else if serverType == "stdio" {
			// If command flag is not set, use connectionStr as command
			cmdStr := serverCommand
			if cmdStr == "" {
				cmdStr = connectionStr
			}
			
			command := []string{cmdStr}
			command = append(command, serverArgs...)
			
			transport = mcp.NewStdioTransport(command, serverEnv)
			server = tool.Server{
				Name:    name,
				Type:    "stdio",
				Command: command,
				Config:  map[string]any{"env": serverEnv},
			}
		} else {
			fmt.Printf("Error: Unsupported server type: %s\n", serverType)
			return
		}

		fmt.Printf("Connecting to MCP server '%s' (%s)...\n", name, serverType)

		// 1. Connect and fetch tools
		client := mcp.NewClient(transport)
		// Ensure we close transport if it's stdio (though for add we just run once)
		defer client.Close()

		initRes, err := client.Initialize(ctx)
		if err != nil {
			fmt.Printf("Error initializing MCP connection: %v\n", err)
			return
		}
		fmt.Printf("Connected to server: %s (%s)\n", initRes.ServerInfo.Name, initRes.ServerInfo.Version)

		mcpTools, err := client.ListTools(ctx)
		if err != nil {
			fmt.Printf("Error listing tools from server: %v\n", err)
			return
		}

		if len(mcpTools) == 0 {
			fmt.Println("Warning: Server returned no tools.")
		}

		// 2. Map MCP tools to our Tool struct
		var tools []tool.Tool
		for _, t := range mcpTools {
			tools = append(tools, tool.Tool{
				Name:        t.Name,
				Description: t.Description,
				InputSchema: t.InputSchema,
			})
		}
		// server.Tools field removed


		// 3. Load Store
		path, err := GetToolsFilePath()
		if err != nil {
			fmt.Printf("Error determining tools file path: %v\n", err)
			return
		}

		store, err := tool.Load(path)
		if err != nil {
			fmt.Printf("Error loading tools from %s: %v\n", path, err)
			return
		}

		// 4. Update Store using Name as key? 
		// Previous implementation used URL as key.
		// We should probably switch to Name as key, but for backward compatibility with existing tests/file format
		// which expected URL as key, we need to be careful.
		// The store map is map[string]Server.
		// If we use Name as key, we should ensure uniqueness.
		
		// Let's use Name as the key.
		// Check if key exists
		if _, exists := store.Servers[name]; exists {
			fmt.Printf("Server '%s' already exists. Updating configuration...\n", name)
		}
		
		store.Servers[name] = server

		if err := store.Save(); err != nil {
			fmt.Printf("Error saving tools: %v\n", err)
			return
		}

		fmt.Printf("Server '%s' added/updated successfully with %d tools.\n", name, len(tools))
		
		// 5. Display tools
		w := tabwriter.NewWriter(os.Stdout, 0, 0, 2, ' ', 0)
		fmt.Fprintln(w, "NAME\tDESCRIPTION")
		for _, t := range tools {
			fmt.Fprintf(w, "%s\t%s\n", t.Name, t.Description)
		}
		w.Flush()
	},
}

func init() {
	serverAddCmd.Flags().StringVar(&serverType, "type", "http", "Server type: 'http' or 'stdio'")
	serverAddCmd.Flags().StringVar(&serverCommand, "command", "", "Command to execute for stdio server")
	serverAddCmd.Flags().StringSliceVar(&serverArgs, "args", []string{}, "Arguments for the stdio command")
	serverAddCmd.Flags().StringSliceVar(&serverEnv, "env", []string{}, "Environment variables for the stdio server (KEY=VALUE)")
	
	ToolCmd.AddCommand(serverAddCmd)
}
