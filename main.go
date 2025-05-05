package main

import (
	"bufio"
	"bytes"
	"encoding/json"
	"fmt"
	"os"
	"strconv"
	"strings"
	"sync"
)

const (
	millisPerHour   = 3600000
	millisPerMinute = 60000
	millisPerSecond = 1000
)

var digits = [10]byte{'0', '1', '2', '3', '4', '5', '6', '7', '8', '9'}

type DraftContent struct {
	Materials struct {
		Texts []TextMaterial `json:"texts"`
	} `json:"materials"`
	Tracks []Track `json:"tracks"`
}

type TextMaterial struct {
	ID      string `json:"id"`
	Content string `json:"content"`
	Words   []Word `json:"words"`
}

type Word struct {
	Begin int64  `json:"begin"`
	End   int64  `json:"end"`
	Text  string `json:"text"`
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

var timeBufferPool = sync.Pool{
	New: func() interface{} {
		return new([12]byte)
	},
}

func formatTime(microseconds int64) string {
	milliseconds := microseconds / 1000
	if milliseconds < 0 {
		milliseconds = 0
	}

	buf := timeBufferPool.Get().(*[12]byte)
	defer timeBufferPool.Put(buf)

	hours := milliseconds / millisPerHour
	milliseconds -= hours * millisPerHour
	minutes := milliseconds / millisPerMinute
	milliseconds -= minutes * millisPerMinute
	seconds := milliseconds / millisPerSecond
	ms := milliseconds - seconds*millisPerSecond

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

func cleanText(input string) string {
	if len(input) == 0 {
		return input
	}

	var sb strings.Builder
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

func buildTextMap(texts []TextMaterial) map[string]TextMaterial {
	textMap := make(map[string]TextMaterial, len(texts))
	for _, text := range texts {
		textMap[text.ID] = text
	}
	return textMap
}

func readDraft(filename string) (DraftContent, error) {
	file, err := os.Open(filename)
	if err != nil {
		return DraftContent{}, fmt.Errorf("failed to open file: %w", err)
	}
	defer file.Close()

	var content DraftContent
	if err := json.NewDecoder(bufio.NewReader(file)).Decode(&content); err != nil {
		return DraftContent{}, fmt.Errorf("failed to parse JSON: %w", err)
	}
	return content, nil
}

func createSubtitles(tracks []Track, textMap map[string]TextMaterial) *bytes.Buffer {
	var buffer = bytes.NewBuffer(nil)
	var subtitleIndex = 1

	for _, track := range tracks {
		if track.Type != "text" {
			continue
		}

		for _, segment := range track.Segments {
			textMaterial, found := textMap[segment.MaterialID]
			if !found {
				continue
			}

			if len(textMaterial.Words) > 0 {
				for _, word := range textMaterial.Words {
					writeSubtitle(buffer, subtitleIndex, word.Begin, word.End, word.Text)
					subtitleIndex++
				}
			} else {
				startTime := segment.TargetTimerange.Start
				endTime := startTime + segment.TargetTimerange.Duration
				writeSubtitle(buffer, subtitleIndex, startTime, endTime, textMaterial.Content)
				subtitleIndex++
			}
		}
	}

	return buffer
}

func writeSubtitle(buffer *bytes.Buffer, index int, startTime int64, endTime int64, content string) {
	buffer.WriteString(strconv.Itoa(index))
	buffer.WriteByte('\n')
	buffer.WriteString(formatTime(startTime))
	buffer.WriteString(" --> ")
	buffer.WriteString(formatTime(endTime))
	buffer.WriteByte('\n')
	buffer.WriteString(cleanText(content))
	buffer.WriteString("\n\n")
}

func main() {
	filePath, err := os.ReadFile("file-path.txt")
	if err != nil {
		fmt.Println("Error reading file path:", err)
		return
	}

	filePath = bytes.TrimSpace(filePath)
	if len(filePath) == 0 {
		fmt.Println("Empty file path")
		return
	}

	draft, err := readDraft(string(filePath))
	if err != nil {
		fmt.Println("Error reading draft:", err)
		return
	}

	textMap := buildTextMap(draft.Materials.Texts)
	subtitles := createSubtitles(draft.Tracks, textMap)

	if err := os.WriteFile("subtitles.srt", subtitles.Bytes(), 0644); err != nil {
		fmt.Println("Error writing subtitles:", err)
		return
	}

	fmt.Println("Subtitles created successfully")
}
