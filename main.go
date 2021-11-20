package main

import (
	_ "embed"
	"fmt"
	"github.com/spf13/cobra"
	"os"
	"text/template"
	"text/template/parse"
)

func main() {
	if err := run(); err != nil {
		fmt.Println(err)
		os.Exit(1)
	}
}

func run() error {
	if len(os.Args) < 2 {
		return fmt.Errorf("you must pass in a template file as the first argument")
	}

	// Process template
	bs, err := os.ReadFile(os.Args[1])
	if err != nil {
		return fmt.Errorf("read template: %w", err)
	}
	tmpl, err := template.New("tmpl").Parse(string(bs))
	if err != nil {
		return fmt.Errorf("parsing template: %w", err)
	}

	// Figure out which variables must be set
	vars := map[string]struct{}{}
	var ns []parse.Node
	ns = append (ns, tmpl.Root.Nodes...)
	for len(ns) > 0 {
		next := ns[0]
		ns = ns[1:]
		switch next.(type) {
		case *parse.ListNode:
			ns = append(ns, next.(*parse.ListNode).Nodes...)
		case *parse.ActionNode:
			for _, cmd := range next.(*parse.ActionNode).Pipe.Cmds {
				for _, arg := range cmd.Args {
					if field, ok := arg.(*parse.FieldNode); ok {
						vars[field.Ident[0]] = struct{}{}
					}
				}
			}
		}
		next.Type()
	}

	// Configure the cli
	cmd := cobra.Command{Use: "tmpl",
		Short: "formats the template with the given variables",
		RunE: func(cmd *cobra.Command, args []string) error {
			vals := map[string]string{}
			for v := range vars {
				lookup := cmd.Flags().Lookup(v)
				if lookup == nil {
					return fmt.Errorf("cant find %v", v)
				}
				vals[v] = lookup.Value.String()
			}
			err = tmpl.Execute(os.Stdout, vals)
			if err != nil {
				return fmt.Errorf("executing template: %w", err)
			}
			return nil
		}}
	for v := range vars {
		cmd.Flags().String(v, "", fmt.Sprintf("replaces the '%v' variable", v))
		err = cmd.MarkFlagRequired(v)
		if err != nil {
			return fmt.Errorf("marking %v required: %w", v, err)
		}
	}
	// remove file argument
	os.Args = append(os.Args[:1], os.Args[2:]...)
	return cmd.Execute()
}