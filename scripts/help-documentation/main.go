package main

import (
	"bytes"
	"context"
	_ "embed"
	"fmt"
	"io"
	"maps"
	"os"
	"path/filepath"
	"slices"
	"strings"
	"text/template"

	"github.com/grafana/grafana-image-renderer/cmd"
	"github.com/urfave/cli/v3"
)

//go:embed help.md.tmpl
var helpTemplate string

type templateData struct {
	Help string
}

func main() {
	if err := run(); err != nil {
		fmt.Println("application error:", err)
		os.Exit(1)
	}
}

func run() error {
	rootCmd := cmd.NewRootCmd()
	rootCmd.Writer = io.Discard
	rootCmd.ErrWriter = io.Discard
	_ = rootCmd.Run(context.Background(), []string{"--help"}) // make urfave/cli fix the command graph
	serverCmd := rootCmd.Command("server")
	if serverCmd == nil {
		return fmt.Errorf("server command not found?")
	}

	help, err := listFlags(serverCmd)
	if err != nil {
		return fmt.Errorf("failed to get custom help for server command: %w", err)
	}

	dat := templateData{
		Help: help,
	}
	tmpl, err := template.New("help").Parse(helpTemplate)
	if err != nil {
		return fmt.Errorf("failed to parse help template: %w", err)
	}
	buffer := &bytes.Buffer{}
	err = tmpl.Execute(buffer, &dat)
	if err != nil {
		return fmt.Errorf("failed to execute template: %w", err)
	}

	gitRoot, err := findGitRoot()
	if err != nil {
		return fmt.Errorf("failed to find git root: %w", err)
	}

	helpFile := filepath.Join(gitRoot, "docs", "sources", "flags.md")
	if err := os.WriteFile(helpFile, buffer.Bytes(), 0644); err != nil {
		return fmt.Errorf("failed to write help file to %q: %w", helpFile, err)
	}

	fmt.Println("Help documentation generated at", helpFile)
	return nil
}

func findGitRoot() (string, error) {
	dir, err := os.Getwd()
	if err != nil {
		return "", fmt.Errorf("failed to get working directory: %w", err)
	}
	for {
		if _, err := os.Stat(filepath.Join(dir, ".git")); err == nil {
			return dir, nil
		}
		dir = filepath.Dir(dir)
		if dir == "/" {
			return "", fmt.Errorf("git root not found")
		}
	}
}

func listFlags(cmd *cli.Command) (string, error) {
	flags := map[string]string{}
	for _, flag := range slices.Concat(cmd.VisibleFlags(), cmd.VisiblePersistentFlags()) {
		names := flag.Names()
		slices.Sort(names)
		key := strings.Join(names, " / ")
		if _, ok := flags[key]; ok {
			continue
		}

		s, err := flagString(flag)
		if err != nil {
			return "", err
		}

		flags[key] = s
	}
	return strings.Join(slices.Sorted(maps.Values(flags)), "\n"), nil
}

func flagString(flag cli.Flag) (string, error) {
	docFlag, ok := flag.(cli.DocGenerationFlag)
	if !ok {
		return "", fmt.Errorf("flag %v is not a doc generation flag", flag.Names())
	}

	var val string
	if docFlag.TakesValue() {
		val = fmt.Sprintf("=<%s>", docFlag.TypeName())
	}

	var names []string
	for _, name := range flag.Names() {
		if len(name) == 1 {
			names = append(names, fmt.Sprintf("-%s%s", name, val))
		} else {
			names = append(names, fmt.Sprintf("--%s%s", name, val))
		}
	}
	slices.Sort(names)

	var envVars []string
	for _, envVar := range docFlag.GetEnvVars() {
		envVars = append(envVars, fmt.Sprintf("${%s}", envVar))
	}
	slices.Sort(envVars)

	var defaultText string
	if docFlag.IsDefaultVisible() && docFlag.GetValue() != "" {
		defaultText = fmt.Sprintf(" [default: %v]", docFlag.GetValue())
	}

	namesStr := strings.Join(names, " / ")
	var envVarsStr string
	if len(envVars) > 0 {
		envVarsStr = fmt.Sprintf(" [%s]", strings.Join(envVars, ", "))
	}

	return fmt.Sprintf("%s%s%s\n    %s", namesStr, defaultText, envVarsStr, docFlag.GetUsage()), nil
}
