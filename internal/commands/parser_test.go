package commands

import "testing"

func TestParse(t *testing.T) {
	tests := []struct {
		name   string
		body   string
		prefix string
		want   Type
	}{
		{name: "start command", body: "!startpal", prefix: "!", want: StartPal},
		{name: "stop command", body: "!stoppal", prefix: "!", want: StopPal},
		{name: "with spaces", body: "   !startpal  ", prefix: "!", want: StartPal},
		{name: "custom prefix", body: "$stoppal", prefix: "$", want: StopPal},
		{name: "unknown", body: "!ping", prefix: "!", want: Unknown},
		{name: "no prefix", body: "stoppal", prefix: "!", want: Unknown},
		{name: "empty", body: "", prefix: "!", want: Unknown},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := Parse(tt.body, tt.prefix)
			if got.Type != tt.want {
				t.Fatalf("Parse(%q) got %v want %v", tt.body, got.Type, tt.want)
			}
		})
	}
}
