package rcon

import "strings"

func ParseShowPlayers(response string) []string {
	lines := strings.Split(response, "\n")
	clean := make([]string, 0, len(lines))
	for _, line := range lines {
		line = strings.TrimSpace(line)
		if line == "" {
			continue
		}
		clean = append(clean, line)
	}
	if len(clean) <= 1 {
		return nil
	}

	players := make([]string, 0, len(clean)-1)
	for _, line := range clean[1:] {
		parts := strings.Split(line, ",")
		if len(parts) == 0 {
			continue
		}
		name := strings.TrimSpace(parts[0])
		if name != "" {
			players = append(players, name)
		}
	}
	return players
}
