package main

import (
	"bufio"
	"encoding/json"
	"fmt"
	"os"
	"strings"
	"sync"
	"time"
)

// Constants for time conversion (precomputed to avoid division)
const (
	millisPerHour   = 3600000
	millisPerMinute = 60000
	millisPerSecond = 1000
)

// digits lookup table for fast number formatting
var digits = [10]byte{'0', '1', '2', '3', '4', '5', '6', '7', '8', '9'}

// srtTimeBufferPool for reusing time format buffers
var srtTimeBufferPool = sync.Pool{
	New: func() interface{} {
		return new([12]byte)
	},
}

// toSRTTime converts microseconds to SRT time format (HH:MM:SS,mmm)
func toSRTTime(microseconds int64) string {
	milliseconds := microseconds / 1000
	if milliseconds < 0 {
		milliseconds = 0
	}

	// Division via multiplication and shifts
	hours := milliseconds / millisPerHour
	milliseconds -= hours * millisPerHour
	minutes := milliseconds / millisPerMinute
	milliseconds -= minutes * millisPerMinute
	seconds := milliseconds / millisPerSecond
	ms := milliseconds - seconds*millisPerSecond

	// Get buffer from pool
	buf := srtTimeBufferPool.Get().(*[12]byte)
	defer srtTimeBufferPool.Put(buf)

	// Format directly into buffer
	buf[0] = digits[hours/10]
	buf[1] = digits[hours%10]
	buf[2] = ':'
	buf[3] = digits[minutes/10]
	buf[4] = digits[minutes%10]
	buf[5] = ':'
	buf[6] = digits[seconds/10]
	buf[7] = digits[seconds%10]
	buf[8] = ','
	buf[9] = digits[ms/100]
	buf[10] = digits[(ms/10)%10]
	buf[11] = digits[ms%10]

	return string(buf[:])
}

// extractText cleans input text by removing brackets, HTML tags, and entities
func extractText(input string) string {
	if len(input) == 0 {
		return input
	}

	// Use stack allocation for small strings
	if len(input) <= 256 {
		var buf [256]byte
		pos := 0
		inTag := false

		for i := 0; i < len(input); {
			switch input[i] {
			case '<':
				inTag = true
				i++
			case '>':
				inTag = false
				i++
			case '[', ']':
				i++
			case '&':
				// Check for common HTML entities with bounds checking
				if i+3 < len(input) && input[i+1] == 'l' && input[i+2] == 't' && input[i+3] == ';' {
					buf[pos] = '<'
					pos++
					i += 4
				} else if i+3 < len(input) && input[i+1] == 'g' && input[i+2] == 't' && input[i+3] == ';' {
					buf[pos] = '>'
					pos++
					i += 4
				} else if i+4 < len(input) && input[i+1] == 'a' && input[i+2] == 'm' && input[i+3] == 'p' && input[i+4] == ';' {
					buf[pos] = '&'
					pos++
					i += 5
				} else if i+5 < len(input) && input[i+1] == 'q' && input[i+2] == 'u' && input[i+3] == 'o' && input[i+4] == 't' && input[i+5] == ';' {
					buf[pos] = '"'
					pos++
					i += 6
				} else if i+4 < len(input) && input[i+1] == '#' && input[i+2] == '3' && input[i+3] == '9' && input[i+4] == ';' {
					buf[pos] = '\''
					pos++
					i += 5
				} else if i+5 < len(input) && input[i+1] == 'n' && input[i+2] == 'b' && input[i+3] == 's' && input[i+4] == 'p' && input[i+5] == ';' {
					buf[pos] = ' '
					pos++
					i += 6
				} else {
					buf[pos] = input[i]
					pos++
					i++
				}
			default:
				if !inTag {
					buf[pos] = input[i]
					pos++
				}
				i++
			}
		}
		return string(buf[:pos])
	}

	// Fallback to strings.Builder for larger strings
	var sb strings.Builder
	sb.Grow(len(input)) // Preallocate to avoid reallocations

	inTag := false
	for i := 0; i < len(input); {
		switch input[i] {
		case '<':
			inTag = true
			i++
		case '>':
			inTag = false
			i++
		case '[', ']':
			i++
		case '&':
			if i+3 < len(input) && input[i+1] == 'l' && input[i+2] == 't' && input[i+3] == ';' {
				sb.WriteByte('<')
				i += 4
			} else if i+3 < len(input) && input[i+1] == 'g' && input[i+2] == 't' && input[i+3] == ';' {
				sb.WriteByte('>')
				i += 4
			} else if i+4 < len(input) && input[i+1] == 'a' && input[i+2] == 'm' && input[i+3] == 'p' && input[i+4] == ';' {
				sb.WriteByte('&')
				i += 5
			} else if i+5 < len(input) && input[i+1] == 'q' && input[i+2] == 'u' && input[i+3] == 'o' && input[i+4] == 't' && input[i+5] == ';' {
				sb.WriteByte('"')
				i += 6
			} else if i+4 < len(input) && input[i+1] == '#' && input[i+2] == '3' && input[i+3] == '9' && input[i+4] == ';' {
				sb.WriteByte('\'')
				i += 5
			} else if i+5 < len(input) && input[i+1] == 'n' && input[i+2] == 'b' && input[i+3] == 's' && input[i+4] == 'p' && input[i+5] == ';' {
				sb.WriteByte(' ')
				i += 6
			} else {
				sb.WriteByte(input[i])
				i++
			}
		default:
			if !inTag {
				sb.WriteByte(input[i])
			}
			i++
		}
	}

	return sb.String()
}

// StringBuilder pool with sync.Pool for better performance
var stringBuilderPool = sync.Pool{
	New: func() interface{} {
		b := make([]byte, 0, 8192) // Larger initial capacity
		return &b
	},
}

func getStringBuilder() *[]byte {
	return stringBuilderPool.Get().(*[]byte)
}

func putStringBuilder(b *[]byte) {
	*b = (*b)[:0]
	stringBuilderPool.Put(b)
}

type DraftContent struct {
	Materials struct {
		Texts []TextMaterial `json:"texts"`
	} `json:"materials"`
	Tracks []Track `json:"tracks"`
}

type TextMaterial struct {
	ID      string `json:"id"`
	Content string `json:"content"`
	Type    string `json:"type"`
	Words   []Word `json:"words"`
}

type Word struct {
	Begin  int64  `json:"begin"`
	End    int64  `json:"end"`
	Text   string `json:"text"`
	Style  int    `json:"style"`
	TextID string `json:"text_id"`
}

type Track struct {
	Type     string    `json:"type"`
	Segments []Segment `json:"segments"`
}

type Segment struct {
	MaterialID      string    `json:"material_id"`
	TargetTimerange Timerange `json:"target_timerange"`
}

type Timerange struct {
	Start    int64 `json:"start"`
	Duration int64 `json:"duration"`
}

// buildTextMaterialMap creates a map for efficient lookup
func buildTextMaterialMap(texts []TextMaterial) map[string]TextMaterial {
	textMap := make(map[string]TextMaterial, len(texts))
	for i := range texts {
		textMap[texts[i].ID] = texts[i]
	}
	return textMap
}

// readJSON reads and parses the JSON file with minimal allocations
func readJSON(filename string) (DraftContent, error) {
	file, err := os.Open(filename)
	if err != nil {
		return DraftContent{}, fmt.Errorf("failed to open file: %w", err)
	}
	defer file.Close()

	var content DraftContent
	dec := json.NewDecoder(bufio.NewReader(file)) // Buffered reading
	dec.UseNumber()                               // More precise number handling
	if err := dec.Decode(&content); err != nil {
		return DraftContent{}, fmt.Errorf("failed to unmarshal JSON: %w", err)
	}
	return content, nil
}

// writeSRT writes the SRT formatted subtitles to a file with direct I/O
func writeSRT(filename string, tracks []Track, textMap map[string]TextMaterial, jsonFilename string) error {
	file, err := os.Create(filename)
	if err != nil {
		return fmt.Errorf("failed to create SRT file: %w", err)
	}
	defer file.Close()

	writer := bufio.NewWriterSize(file, 64*1024) // 64KB buffer
	defer writer.Flush()

	subtitleIndex := 1
	buf := getStringBuilder()
	defer putStringBuilder(buf)

	// Reusable byte slices for common patterns
	arrow := []byte(" --> ")
	newline := []byte("\n")
	emptyLine := []byte("\n\n")

	// Preallocate index buffer to avoid allocations in loop
	var indexBuf [12]byte
	indexStart := 8 // Start position for index digits

	for _, track := range tracks {
		if track.Type != "text" {
			continue
		}

		for _, segment := range track.Segments {
			textMaterial, found := textMap[segment.MaterialID]
			if !found {
				fmt.Printf("Warning: Text material with ID %s not found in '%s'\n", segment.MaterialID, jsonFilename)
				continue
			}

			if len(textMaterial.Words) > 0 {
				for _, word := range textMaterial.Words {
					// Format index directly into preallocated buffer
					n := subtitleIndex
					pos := indexStart
					for n > 0 {
						indexBuf[pos] = digits[n%10]
						n /= 10
						pos--
					}
					indexSlice := indexBuf[pos+1 : indexStart+1]

					// Format SRT entry
					*buf = append((*buf)[:0], indexSlice...)
					*buf = append(*buf, newline...)
					*buf = append(*buf, toSRTTime(word.Begin)...)
					*buf = append(*buf, arrow...)
					*buf = append(*buf, toSRTTime(word.End)...)
					*buf = append(*buf, newline...)
					*buf = append(*buf, extractText(word.Text)...)
					*buf = append(*buf, emptyLine...)

					if _, err := writer.Write(*buf); err != nil {
						return fmt.Errorf("failed to write SRT entry: %w", err)
					}
					subtitleIndex++
				}
			} else {
				// Format index directly into preallocated buffer
				n := subtitleIndex
				pos := indexStart
				for n > 0 {
					indexBuf[pos] = digits[n%10]
					n /= 10
					pos--
				}
				indexSlice := indexBuf[pos+1 : indexStart+1]

				// Format SRT entry
				start := segment.TargetTimerange.Start
				end := start + segment.TargetTimerange.Duration
				*buf = append((*buf)[:0], indexSlice...)
				*buf = append(*buf, newline...)
				*buf = append(*buf, toSRTTime(start)...)
				*buf = append(*buf, arrow...)
				*buf = append(*buf, toSRTTime(end)...)
				*buf = append(*buf, newline...)
				*buf = append(*buf, extractText(textMaterial.Content)...)
				*buf = append(*buf, emptyLine...)

				if _, err := writer.Write(*buf); err != nil {
					return fmt.Errorf("failed to write SRT entry: %w", err)
				}
				subtitleIndex++
			}
		}
	}

	return nil
}

var (
	version = "dev"
	commit  = "commit"
	date    = "date"
)

func main() {
	// Print version info
	fmt.Printf("Version: %s\nCommit Hash: %s\nBuild Date: %s\n", version, commit, date)

	// Read file path with direct byte access and optimized trimming
	filePathBytes, err := os.ReadFile("file-path.txt")
	if err != nil {
		fmt.Println("Error reading configuration file 'file-path.txt':", err)
		return
	}

	// Trim whitespace efficiently using byte scanning
	start, end := 0, len(filePathBytes)
	for start < end && filePathBytes[start] <= ' ' {
		start++
	}
	for end > start && filePathBytes[end-1] <= ' ' {
		end--
	}

	if start == end {
		fmt.Println("Error: 'file-path.txt' is empty or contains only whitespace.")
		return
	}
	jsonFilename := string(filePathBytes[start:end])

	// Read and parse JSON
	draftContent, err := readJSON(jsonFilename)
	if err != nil {
		fmt.Println(err)
		return
	}

	// Build text material map
	textMap := buildTextMaterialMap(draftContent.Materials.Texts)

	// Generate SRT filename using time-based suffix
	now := time.Now()
	randomSuffix := now.UnixNano() % 10_000_000_000
	srtFilename := fmt.Sprintf("subtitles-%d.srt", randomSuffix)

	// Convert and write SRT
	err = writeSRT(srtFilename, draftContent.Tracks, textMap, jsonFilename)
	if err != nil {
		fmt.Println(err)
		return
	}

	fmt.Printf("Successfully converted subtitles from '%s' to %s\n", jsonFilename, srtFilename)
}
