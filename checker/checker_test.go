package checker

import (
	"testing"

	"github.com/VidarTessem/tglys/config"
)

func TestParseState_Array(t *testing.T) {
	tests := []struct {
		name    string
		body    string
		want    State
		wantErr bool
	}{
		{
			name: "array with red",
			body: `[{"trafikklys":"red"}]`,
			want: StateRed,
		},
		{
			name: "array with yellow",
			body: `[{"trafikklys":"yellow"}]`,
			want: StateYellow,
		},
		{
			name: "array with green",
			body: `[{"trafikklys":"green"}]`,
			want: StateGreen,
		},
		{
			name: "array with extra fields",
			body: `[{"id":1,"trafikklys":"green","other":"value"}]`,
			want: StateGreen,
		},
		{
			name: "array – first item wins",
			body: `[{"trafikklys":"red"},{"trafikklys":"green"}]`,
			want: StateRed,
		},
		{
			name: "array – skip empty, pick next",
			body: `[{"trafikklys":""},{"trafikklys":"yellow"}]`,
			want: StateYellow,
		},
		{
			name:    "array – no trafikklys key",
			body:    `[{"other":"value"}]`,
			wantErr: true,
		},
		{
			name:    "empty array",
			body:    `[]`,
			wantErr: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, err := parseState([]byte(tt.body))
			if tt.wantErr {
				if err == nil {
					t.Errorf("expected error, got state=%q", got)
				}
				return
			}
			if err != nil {
				t.Fatalf("unexpected error: %v", err)
			}
			if got != tt.want {
				t.Errorf("got %q, want %q", got, tt.want)
			}
		})
	}
}

func TestParseState_Object(t *testing.T) {
	tests := []struct {
		name    string
		body    string
		want    State
		wantErr bool
	}{
		{
			name: "object red",
			body: `{"trafikklys":"red"}`,
			want: StateRed,
		},
		{
			name: "object yellow",
			body: `{"trafikklys":"yellow"}`,
			want: StateYellow,
		},
		{
			name: "object green",
			body: `{"trafikklys":"green"}`,
			want: StateGreen,
		},
		{
			name:    "object missing key",
			body:    `{"other":"value"}`,
			wantErr: true,
		},
		{
			name:    "invalid JSON",
			body:    `not json`,
			wantErr: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, err := parseState([]byte(tt.body))
			if tt.wantErr {
				if err == nil {
					t.Errorf("expected error, got state=%q", got)
				}
				return
			}
			if err != nil {
				t.Fatalf("unexpected error: %v", err)
			}
			if got != tt.want {
				t.Errorf("got %q, want %q", got, tt.want)
			}
		})
	}
}

func TestParseState_UnknownValue(t *testing.T) {
	got, err := parseState([]byte(`{"trafikklys":"purple"}`))
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if got != StateUnknown {
		t.Errorf("got %q, want %q", got, StateUnknown)
	}
}

func TestVolumeForState(t *testing.T) {
	cfg := config.Config{
		RedVolume:    0,
		YellowVolume: 50,
		GreenVolume:  100,
	}
	tests := []struct {
		state State
		want  int
	}{
		{StateRed, 0},
		{StateYellow, 50},
		{StateGreen, 100},
		{StateUnknown, 100}, // falls back to GreenVolume
	}
	for _, tt := range tests {
		got := volumeForState(tt.state, cfg)
		if got != tt.want {
			t.Errorf("volumeForState(%q) = %d, want %d", tt.state, got, tt.want)
		}
	}
}

func TestNormalise(t *testing.T) {
	if got := normalise("red"); got != StateRed {
		t.Errorf("normalise(red) = %q", got)
	}
	if got := normalise("green"); got != StateGreen {
		t.Errorf("normalise(green) = %q", got)
	}
	if got := normalise("yellow"); got != StateYellow {
		t.Errorf("normalise(yellow) = %q", got)
	}
	if got := normalise("other"); got != StateUnknown {
		t.Errorf("normalise(other) = %q", got)
	}
}
