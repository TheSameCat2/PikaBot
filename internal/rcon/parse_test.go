package rcon

import (
	"reflect"
	"testing"
)

func TestParseShowPlayers(t *testing.T) {
	tests := []struct {
		name     string
		response string
		want     []string
	}{
		{
			name:     "header only",
			response: "name,playeruid,steamid\n",
			want:     nil,
		},
		{
			name:     "header with spaces and blank lines",
			response: " name,playeruid,steamid \n\n",
			want:     nil,
		},
		{
			name:     "multiple players",
			response: "name,playeruid,steamid\nAlice,uid1,steam1\nBob,uid2,steam2\n",
			want:     []string{"Alice", "Bob"},
		},
		{
			name:     "ignores empty player name",
			response: "name,playeruid,steamid\n,uid1,steam1\nCharlie,uid2,steam2\n",
			want:     []string{"Charlie"},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := ParseShowPlayers(tt.response)
			if !reflect.DeepEqual(got, tt.want) {
				t.Fatalf("ParseShowPlayers() got %v want %v", got, tt.want)
			}
		})
	}
}
