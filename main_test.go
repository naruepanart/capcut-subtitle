package main

import (
	"encoding/json"
	"os"
	"reflect"
	"testing"
)

func TestFormatTime(t *testing.T) {
	tests := []struct {
		name       string
		input      int64
		want       string
		beforeTest func()
	}{
		{
			name:  "zero milliseconds",
			input: 0,
			want:  "00:00:00,000",
		},
		{
			name:  "one hour",
			input: 3600 * 1000 * 1000,
			want:  "01:00:00,000",
		},
		{
			name:  "one minute",
			input: 60 * 1000 * 1000,
			want:  "00:01:00,000",
		},
		{
			name:  "one second",
			input: 1000 * 1000,
			want:  "00:00:01,000",
		},
		{
			name:  "one millisecond",
			input: 1000,
			want:  "00:00:00,001",
		},
		{
			name:  "complex time",
			input: 3723001000, // 1 hour, 2 minutes, 3 seconds, 1 millisecond
			want:  "01:02:03,001",
		},
		{
			name:  "negative time",
			input: -1000,
			want:  "00:00:00,000",
		},
		{
			name:  "max single digit hours",
			input: 9 * 3600 * 1000 * 1000,
			want:  "09:00:00,000",
		},
		{
			name:  "double digit hours",
			input: 10 * 3600 * 1000 * 1000,
			want:  "10:00:00,000",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if tt.beforeTest != nil {
				tt.beforeTest()
			}

			got := formatTime(tt.input)
			if got != tt.want {
				t.Errorf("formatTime() = %v, want %v", got, tt.want)
			}
		})
	}

	// Test pool reuse
	t.Run("pool reuse", func(t *testing.T) {
		initialPoolSize := timeBufferPool.New().(*[12]byte)
		timeBufferPool.Put(initialPoolSize)

		formatTime(1000)
		formatTime(2000)

		// Verify pool is being used by checking if the same buffer is reused
		buf1 := timeBufferPool.Get().(*[12]byte)
		timeBufferPool.Put(buf1)
		buf2 := timeBufferPool.Get().(*[12]byte)
		if buf1 != buf2 {
			t.Error("Expected buffer pool to reuse buffers")
		}
	})
}

func TestCleanText(t *testing.T) {
	tests := []struct {
		name  string
		input string
		want  string
	}{
		{
			name:  "empty string",
			input: "",
			want:  "",
		},
		{
			name:  "no tags",
			input: "Hello world",
			want:  "Hello world",
		},
		{
			name:  "with tags",
			input: "Hello <b>world</b>",
			want:  "Hello world",
		},
		{
			name:  "with square brackets",
			input: "Hello [world]",
			want:  "Hello world",
		},
		{
			name:  "with escaped lt",
			input: "Hello &lt;world&gt;",
			want:  "Hello <world>",
		},
		{
			name:  "with escaped gt",
			input: "Hello &gt;world&lt;",
			want:  "Hello >world<",
		},
		{
			name:  "mixed content",
			input: "<b>Hello</b> [world] &lt;3",
			want:  "Hello world <3",
		},
		{
			name:  "multiple tags",
			input: "<i><b>Hello</b> world</i>",
			want:  "Hello world",
		},
		{
			name:  "self-closing tag",
			input: "Hello<br/>world",
			want:  "Helloworld",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := cleanText(tt.input)
			if got != tt.want {
				t.Errorf("cleanText() = %v, want %v", got, tt.want)
			}
		})
	}
}

func TestBuildTextMap(t *testing.T) {
	tests := []struct {
		name  string
		input []TextMaterial
		want  map[string]TextMaterial
	}{
		{
			name:  "empty slice",
			input: []TextMaterial{},
			want:  map[string]TextMaterial{},
		},
		{
			name: "single text material",
			input: []TextMaterial{
				{ID: "1", Content: "Hello"},
			},
			want: map[string]TextMaterial{
				"1": {ID: "1", Content: "Hello"},
			},
		},
		{
			name: "multiple text materials",
			input: []TextMaterial{
				{ID: "1", Content: "Hello"},
				{ID: "2", Content: "World"},
			},
			want: map[string]TextMaterial{
				"1": {ID: "1", Content: "Hello"},
				"2": {ID: "2", Content: "World"},
			},
		},
		{
			name: "duplicate IDs (should overwrite)",
			input: []TextMaterial{
				{ID: "1", Content: "Hello"},
				{ID: "1", Content: "World"},
			},
			want: map[string]TextMaterial{
				"1": {ID: "1", Content: "World"},
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := buildTextMap(tt.input)
			if !reflect.DeepEqual(got, tt.want) {
				t.Errorf("buildTextMap() = %v, want %v", got, tt.want)
			}
		})
	}
}

func TestReadDraft(t *testing.T) {
	// Create a temporary file for testing
	tempFile, err := os.CreateTemp("", "test-draft-*.json")
	if err != nil {
		t.Fatal(err)
	}
	defer os.Remove(tempFile.Name())

	// Write test data to the temporary file
	testDraft := DraftContent{
		Materials: struct {
			Texts []TextMaterial `json:"texts"`
		}{
			Texts: []TextMaterial{
				{ID: "1", Content: "Test content"},
			},
		},
		Tracks: []Track{
			{Type: "text", Segments: []Segment{
				{MaterialID: "1", TargetTimerange: Timerange{Start: 0, Duration: 1000}},
			}},
		},
	}

	if err := json.NewEncoder(tempFile).Encode(testDraft); err != nil {
		t.Fatal(err)
	}
	tempFile.Close()

	tests := []struct {
		name       string
		filename   string
		want       DraftContent
		wantErr    bool
		beforeTest func()
	}{
		{
			name:     "valid file",
			filename: tempFile.Name(),
			want:     testDraft,
			wantErr:  false,
		},
		{
			name:     "non-existent file",
			filename: "nonexistent.json",
			want:     DraftContent{},
			wantErr:  true,
		},
		{
			name: "invalid JSON",
			filename: func() string {
				f, err := os.CreateTemp("", "invalid-json-*.json")
				if err != nil {
					t.Fatal(err)
				}
				f.WriteString("{invalid json}")
				f.Close()
				return f.Name()
			}(),
			want:    DraftContent{},
			wantErr: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if tt.beforeTest != nil {
				tt.beforeTest()
			}

			got, err := readDraft(tt.filename)
			if (err != nil) != tt.wantErr {
				t.Errorf("readDraft() error = %v, wantErr %v", err, tt.wantErr)
				return
			}
			if !tt.wantErr && !reflect.DeepEqual(got, tt.want) {
				t.Errorf("readDraft() = %v, want %v", got, tt.want)
			}
		})
	}
}

func TestCreateSubtitles(t *testing.T) {
	tests := []struct {
		name    string
		tracks  []Track
		textMap map[string]TextMaterial
		want    string
	}{
		{
			name:    "empty inputs",
			tracks:  []Track{},
			textMap: map[string]TextMaterial{},
			want:    "",
		},
		{
			name: "text track with words",
			tracks: []Track{
				{
					Type: "text",
					Segments: []Segment{
						{
							MaterialID: "1",
							TargetTimerange: Timerange{
								Start:    1000000,
								Duration: 2000000,
							},
						},
					},
				},
			},
			textMap: map[string]TextMaterial{
				"1": {
					ID:      "1",
					Content: "Full content",
					Words: []Word{
						{Begin: 1000000, End: 1500000, Text: "Hello"},
						{Begin: 1500000, End: 3000000, Text: "world"},
					},
				},
			},
			want: `1
00:00:01,000 --> 00:00:01,500
Hello

2
00:00:01,500 --> 00:00:03,000
world

`,
		},
		{
			name: "text track without words",
			tracks: []Track{
				{
					Type: "text",
					Segments: []Segment{
						{
							MaterialID: "1",
							TargetTimerange: Timerange{
								Start:    1000000,
								Duration: 2000000,
							},
						},
					},
				},
			},
			textMap: map[string]TextMaterial{
				"1": {
					ID:      "1",
					Content: "Hello world",
					Words:   []Word{},
				},
			},
			want: `1
00:00:01,000 --> 00:00:03,000
Hello world

`,
		},
		{
			name: "multiple segments",
			tracks: []Track{
				{
					Type: "text",
					Segments: []Segment{
						{
							MaterialID: "1",
							TargetTimerange: Timerange{
								Start:    1000000,
								Duration: 2000000,
							},
						},
						{
							MaterialID: "2",
							TargetTimerange: Timerange{
								Start:    4000000,
								Duration: 1000000,
							},
						},
					},
				},
			},
			textMap: map[string]TextMaterial{
				"1": {
					ID:      "1",
					Content: "First segment",
					Words:   []Word{},
				},
				"2": {
					ID:      "2",
					Content: "Second segment",
					Words:   []Word{},
				},
			},
			want: `1
00:00:01,000 --> 00:00:03,000
First segment

2
00:00:04,000 --> 00:00:05,000
Second segment

`,
		},
		{
			name: "non-text track ignored",
			tracks: []Track{
				{
					Type: "video",
					Segments: []Segment{
						{
							MaterialID: "1",
							TargetTimerange: Timerange{
								Start:    1000000,
								Duration: 2000000,
							},
						},
					},
				},
				{
					Type: "text",
					Segments: []Segment{
						{
							MaterialID: "2",
							TargetTimerange: Timerange{
								Start:    4000000,
								Duration: 1000000,
							},
						},
					},
				},
			},
			textMap: map[string]TextMaterial{
				"1": {
					ID:      "1",
					Content: "This should be ignored",
					Words:   []Word{},
				},
				"2": {
					ID:      "2",
					Content: "This should be included",
					Words:   []Word{},
				},
			},
			want: `1
00:00:04,000 --> 00:00:05,000
This should be included

`,
		},
		{
			name: "missing material ID",
			tracks: []Track{
				{
					Type: "text",
					Segments: []Segment{
						{
							MaterialID: "1",
							TargetTimerange: Timerange{
								Start:    1000000,
								Duration: 2000000,
							},
						},
						{
							MaterialID: "999", // Doesn't exist in textMap
							TargetTimerange: Timerange{
								Start:    4000000,
								Duration: 1000000,
							},
						},
					},
				},
			},
			textMap: map[string]TextMaterial{
				"1": {
					ID:      "1",
					Content: "This should be included",
					Words:   []Word{},
				},
			},
			want: `1
00:00:01,000 --> 00:00:03,000
This should be included

`,
		},
		{
			name: "text with tags and special characters",
			tracks: []Track{
				{
					Type: "text",
					Segments: []Segment{
						{
							MaterialID: "1",
							TargetTimerange: Timerange{
								Start:    1000000,
								Duration: 2000000,
							},
						},
					},
				},
			},
			textMap: map[string]TextMaterial{
				"1": {
					ID:      "1",
					Content: "<b>Hello</b> &lt;world&gt; [test]",
					Words:   []Word{},
				},
			},
			want: `1
00:00:01,000 --> 00:00:03,000
Hello <world> test

`,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			buf := createSubtitles(tt.tracks, tt.textMap)
			got := buf.String()
			if got != tt.want {
				t.Errorf("createSubtitles() = \n%v\nwant\n%v", got, tt.want)
			}
		})
	}
}
