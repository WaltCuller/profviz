package internal

import (
	"fmt"
	"os"
	"strings"
	"unicode/utf8"

	"github.com/MakeNowJust/heredoc/v2"
	"github.com/fatih/color"
	"github.com/spf13/cobra"
	"github.com/spf13/pflag"
	"golang.org/x/exp/utf8string"
)

var rootCmd = &cobra.Command{
	Use:     "profviz <command> <subcommand> [flags]",
	Aliases: []string{"pvz"},
	Short:   "",
	Long:    "",
	Example: heredoc.Doc(`
		TODO`),
	Annotations: map[string]string{
		"help:feedback": heredoc.Doc(`
			Open an issue at https://github.com/WaltCuller/profviz/issues/new/choose`),
	},
	Version:       version,
	SilenceUsage:  true,
	SilenceErrors: true,
}

// Execute adds all child commands to the root command and sets flags appropriately.
// This is called by main.main().
// It only needs to happen once to the rootCmd.
func Execute() {
	if err := rootCmd.Execute(); err != nil {
		fmt.Println(err)
		os.Exit(1)
	}
}

func isRootCmd(cmd *cobra.Command) bool {
	return cmd != nil && !cmd.HasParent()
}

func printSubcommandSuggestions(cmd *cobra.Command, arg string) {
	out := cmd.OutOrStdout()
	fmt.Fprintf(out, "unknown command %q for %q\n", arg, cmd.CommandPath())
	if cmd.SuggestionsMinimumDistance <= 0 {
		cmd.SuggestionsMinimumDistance = 2
	}
	candidates := cmd.SuggestionsFor(arg)
	if len(candidates) > 0 {
		fmt.Fprint(out, "\nDid you mean this?\n")
		for _, c := range candidates {
			fmt.Fprintf(out, "\t%s\n", c)
		}
	}
	fmt.Fprintln(out)
	rootUsageFunc(cmd)
}

func rootUsageFunc(cmd *cobra.Command) error {
	out := cmd.OutOrStdout()
	fmt.Fprintf(out, "Usage: %s", cmd.UseLine())
	if subcmds := cmd.Commands(); len(subcmds) > 0 {
		fmt.Fprint(out, "\n\nAvailable commands:\n")
		for _, c := range subcmds {
			if c.Hidden {
				continue
			}
			fmt.Fprintf(out, "  %s\n", c.Name())
		}
	}

	var localFlags []*displayLine
	cmd.LocalFlags().VisitAll(func(f *pflag.Flag) {
		localFlags = append(localFlags, &displayLine{name: "--" + f.Name, desc: capitalize(f.Usage)})
	})
	adjustPadding(localFlags...)
	if len(localFlags) > 0 {
		fmt.Fprint(out, "\n\nFlags:\n")
		for _, l := range localFlags {
			fmt.Fprintf(out, "  %s\n", l.String())
		}
	}
	return nil
}

func rootHelpFunc(cmd *cobra.Command, args []string) {
	// Display helpful error message when user mistypes a subcommand.
	if isRootCmd(cmd.Parent()) && len(args) >= 2 && args[1] != "--help" && args[1] != "-h" {
		printSubcommandSuggestions(cmd, args[1])
		return
	}

	var lines []*displayLine
	var commands []*displayLine
	for _, c := range cmd.Commands() {
		if c.Hidden || c.Short == "" || c.Name() == "help" {
			continue
		}
		l := &displayLine{name: c.Name() + ":", desc: capitalize(c.Short)}
		commands = append(commands, l)
		lines = append(lines, l)
	}
	var localFlags []*displayLine
	cmd.LocalFlags().VisitAll(func(f *pflag.Flag) {
		l := &displayLine{name: "--" + f.Name, desc: capitalize(f.Usage)}
		localFlags = append(localFlags, l)
		lines = append(lines, l)
	})
	var inheritedFlags []*displayLine
	cmd.InheritedFlags().VisitAll(func(f *pflag.Flag) {
		l := &displayLine{name: "--" + f.Name, desc: capitalize(f.Usage)}
		inheritedFlags = append(inheritedFlags, l)
		lines = append(lines, l)
	})
	adjustPadding(lines...)

	type helpEntry struct {
		Title string
		Body  string
	}
	var helpEntries []*helpEntry
	desc := cmd.Long
	if desc == "" {
		desc = cmd.Short
	}
	if desc != "" {
		helpEntries = append(helpEntries, &helpEntry{"", desc})
	}
	helpEntries = append(helpEntries, &helpEntry{"USAGE", cmd.UseLine()})
	if len(commands) > 0 {
		helpEntries = append(helpEntries, &helpEntry{"COMMANDS", displayLines(commands).String()})
	}
	if cmd.LocalFlags().HasFlags() {
		helpEntries = append(helpEntries, &helpEntry{"FLAGS", displayLines(localFlags).String()})
	}
	if cmd.InheritedFlags().HasFlags() {
		helpEntries = append(helpEntries, &helpEntry{"INHERITED FLAGS", displayLines(inheritedFlags).String()})
	}
	if cmd.Example != "" {
		helpEntries = append(helpEntries, &helpEntry{"EXAMPLES", cmd.Example})
	}
	helpEntries = append(helpEntries, &helpEntry{"LEARN MORE", heredoc.Doc(`
		Use 'profviz <command> <subcommand> --help' for more information about a command.`)})
	if s, ok := cmd.Annotations["help:feedback"]; ok {
		helpEntries = append(helpEntries, &helpEntry{"FEEDBACK", s})
	}

	out := cmd.OutOrStdout()
	bold := color.New(color.Bold)
	for _, e := range helpEntries {
		if e.Title != "" {
			// If there is a title, add indentation to each line in the body
			bold.Fprintln(out, e.Title)
			fmt.Fprintln(out, indent(e.Body, 2 /* spaces */))
		} else {
			// If there is no title, print the body as is
			fmt.Fprintln(out, e.Body)
		}
		fmt.Fprintln(out)
	}
}

// displayLine represents a line displayed in the output as '<name> <desc>',
// where pad is used to pad the name from desc.
type displayLine struct {
	name string
	desc string
	pad  int // number of rpad
}

func (l *displayLine) String() string {
	return rpad(l.name, l.pad) + l.desc
}

type displayLines []*displayLine

func (dls displayLines) String() string {
	var lines []string
	for _, dl := range dls {
		lines = append(lines, dl.String())
	}
	return strings.Join(lines, "\n")
}

func adjustPadding(lines ...*displayLine) {
	// find the maximum width of the name
	max := 0
	for _, l := range lines {
		if n := utf8.RuneCountInString(l.name); n > max {
			max = n
		}
	}
	for _, l := range lines {
		l.pad = max
	}
}

// rpad adds padding to the right of a string.
func rpad(s string, padding int) string {
	tmpl := fmt.Sprintf("%%-%ds ", padding)
	return fmt.Sprintf(tmpl, s)
}

// Capitalize the first word in the given string.
func capitalize(s string) string {
	str := utf8string.NewString(s)
	if str.RuneCount() == 0 {
		return ""
	}
	var b strings.Builder
	b.WriteString(strings.ToUpper(string(str.At(0))))
	b.WriteString(str.Slice(1, str.RuneCount()))
	return b.String()
}

// indent indents the given text by given spaces.
func indent(text string, space int) string {
	if len(text) == 0 {
		return ""
	}
	var b strings.Builder
	indentation := strings.Repeat(" ", space)
	lastRune := '\n'
	for _, r := range text {
		if lastRune == '\n' {
			b.WriteString(indentation)
		}
		b.WriteRune(r)
		lastRune = r
	}
	return b.String()
}
