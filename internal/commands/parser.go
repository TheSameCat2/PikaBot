package commands

import "strings"

type Type int

const (
	Unknown Type = iota
	StartPal
	StopPal
)

type Command struct {
	Type Type
	Raw  string
}

func Parse(body, prefix string) Command {
	trimmed := strings.TrimSpace(body)
	if trimmed == "" {
		return Command{Type: Unknown, Raw: trimmed}
	}
	if prefix == "" {
		prefix = "!"
	}
	if !strings.HasPrefix(trimmed, prefix) {
		return Command{Type: Unknown, Raw: trimmed}
	}

	fields := strings.Fields(strings.TrimPrefix(trimmed, prefix))
	if len(fields) == 0 {
		return Command{Type: Unknown, Raw: trimmed}
	}

	switch strings.ToLower(fields[0]) {
	case "startpal":
		return Command{Type: StartPal, Raw: trimmed}
	case "stoppal":
		return Command{Type: StopPal, Raw: trimmed}
	default:
		return Command{Type: Unknown, Raw: trimmed}
	}
}
