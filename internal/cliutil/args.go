// Package cliutil contains small CLI argument helpers shared by jutland-sim,
// jutland-duel and jutland-macro.
package cliutil

import "strings"

// Split separates flag tokens from positionals so Go's flag package can parse
// flags that appear after a scenario path. boolFlags lists flags that do not
// consume the next token (e.g. open/mock).
func Split(args []string, boolFlags ...string) (flags, positionals []string) {
	bools := make(map[string]bool, len(boolFlags))
	for _, name := range boolFlags {
		bools[name] = true
	}
	for i := 0; i < len(args); i++ {
		arg := args[i]
		if !strings.HasPrefix(arg, "-") || arg == "-" {
			positionals = append(positionals, arg)
			continue
		}
		flags = append(flags, arg)
		name := strings.TrimLeft(arg, "-")
		if eq := strings.IndexByte(name, '='); eq >= 0 {
			continue
		}
		if bools[name] {
			continue
		}
		// Value flag: consume the next token unless it is another flag.
		if i+1 < len(args) && !strings.HasPrefix(args[i+1], "-") {
			i++
			flags = append(flags, args[i])
		}
	}
	return flags, positionals
}
