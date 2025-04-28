package main

import (
	"bufio"
	"encoding/json"
	"fmt"
	"os"
	"regexp"
	"strconv"
	"strings"
	"time"
)

// toSRTTime converts microseconds to SRT time format (HH:MM:SS,mmm)
func toSRTTime(microseconds int64) string {
	milliseconds := microseconds / 1000
	if milliseconds < 0 {
		milliseconds = 0
	}
	duration := time.Duration(milliseconds) * time.Millisecond

	hours := int(duration.Hours())
	minutes := int(duration.Minutes()) % 60
	seconds := int(duration.Seconds()) % 60
	ms := int(duration.Milliseconds()) % 1000

	return fmt.Sprintf("%02d:%02d:%02d,%03d", hours, minutes, seconds, ms)
}

var htmlTagRegex = regexp.MustCompile(`<[^>]*>`)

// extractText cleans input text by removing brackets, HTML tags, and entities.
func extractText(input string) string {
	input = strings.ReplaceAll(input, "[", "")
	input = strings.ReplaceAll(input, "]", "")
	input = htmlTagRegex.ReplaceAllString(input, "")
	replacements := map[string]string{
		"&lt;":   "<",
		"&gt;":   ">",
		"&amp;":  "&",
		"&quot;": `"`,
		"&#39;":  "'",
		"&nbsp;": " ",
	}
	for entity, replacement := range replacements {
		input = strings.ReplaceAll(input, entity, replacement)
	}
	return input
}

// DraftContent represents the structure of the input JSON.
type DraftContent struct {
	Materials struct {
		Texts []TextMaterial `json:"texts"`
	} `json:"materials"`
	Tracks []Track `json:"tracks"`
}

// TextMaterial contains text content and word timing information.
type TextMaterial struct {
	ID      string `json:"id"`
	Content string `json:"content"`
	Type    string `json:"type"`
	Words   []Word `json:"words"`
}

// Word represents a single word with timing information.
type Word struct {
	Begin  int64  `json:"begin"`
	End    int64  `json:"end"`
	Text   string `json:"text"`
	Style  int    `json:"style"`
	TextID string `json:"text_id"`
}

// Track contains segments of media content.
type Track struct {
	Type     string    `json:"type"`
	Segments []Segment `json:"segments"`
}

// Segment links material to its timing information.
type Segment struct {
	MaterialID      string    `json:"material_id"`
	TargetTimerange Timerange `json:"target_timerange"`
}

// Timerange defines the start and duration of a segment.
type Timerange struct {
	Start    int64 `json:"start"`
	Duration int64 `json:"duration"`
}

// Helper Functions

// buildTextMaterialMap creates a map for efficient lookup of TextMaterial by ID.
func buildTextMaterialMap(texts []TextMaterial) map[string]TextMaterial {
	textMap := make(map[string]TextMaterial, len(texts))
	for _, text := range texts {
		textMap[text.ID] = text
	}
	return textMap
}

// readJSON reads and parses the JSON file.
func readJSON(filename string) (DraftContent, error) {
	data, err := os.ReadFile(filename)
	if err != nil {
		return DraftContent{}, fmt.Errorf("failed to read file: %w", err)
	}

	var content DraftContent
	err = json.Unmarshal(data, &content)
	if err != nil {
		return DraftContent{}, fmt.Errorf("failed to unmarshal JSON: %w", err)
	}
	return content, nil
}

// writeSRT writes the SRT formatted subtitles to a file.
func writeSRT(filename string, tracks []Track, textMap map[string]TextMaterial, jsonFilename string) error { // Added jsonFilename
	file, err := os.Create(filename)
	if err != nil {
		return fmt.Errorf("failed to create SRT file: %w", err)
	}
	defer file.Close()

	writer := bufio.NewWriter(file)
	defer writer.Flush()

	subtitleIndex := 1
	for _, track := range tracks {
		if track.Type != "text" {
			continue
		}
		for _, segment := range track.Segments {
			textMaterial, found := textMap[segment.MaterialID]
			if !found {
				fmt.Printf("Warning: Text material with ID %s not found in '%s'\n", segment.MaterialID, jsonFilename) // use jsonFilename
				continue
			}

			text := extractText(textMaterial.Content)
			var startTime, endTime string

			if len(textMaterial.Words) > 0 {
				for _, word := range textMaterial.Words {
					startTime = toSRTTime(word.Begin)
					endTime = toSRTTime(word.End)
					wordText := extractText(word.Text)
					_, err = fmt.Fprintf(writer, "%d\n%s --> %s\n%s\n\n", subtitleIndex, startTime, endTime, wordText)
					if err != nil {
						return fmt.Errorf("failed to write SRT entry: %w", err)
					}
					subtitleIndex++
				}
			} else {
				startTime = toSRTTime(segment.TargetTimerange.Start)
				endTime = toSRTTime(segment.TargetTimerange.Start + segment.TargetTimerange.Duration)
				_, err = fmt.Fprintf(writer, "%d\n%s --> %s\n%s\n\n", subtitleIndex, startTime, endTime, text)
				if err != nil {
					return fmt.Errorf("failed to write SRT entry: %w", err)
				}
				subtitleIndex++
			}
		}
	}
	return nil
}

var version = "dev"

func main() {
	fmt.Println("App version:", version)

	// Read file path
	filePathBytes, err := os.ReadFile("file-path.txt")
	if err != nil {
		fmt.Println("Error reading configuration file 'file-path.txt':", err)
		fmt.Println("Please ensure 'file-path.txt' exists and contains the name of the JSON file to process.")
		return
	}
	jsonFilename := strings.TrimSpace(string(filePathBytes))
	if jsonFilename == "" {
		fmt.Println("Error: 'file-path.txt' is empty or contains only whitespace.")
		fmt.Println("Please ensure 'file-path.txt' contains the name of the JSON file to process.")
		return
	}

	// Read and parse JSON
	draftContent, err := readJSON(jsonFilename)
	if err != nil {
		fmt.Println(err)
		return
	}

	// Build text material map
	textMap := buildTextMaterialMap(draftContent.Materials.Texts)

	// Generate SRT filename
	randomSuffix := strconv.FormatInt(time.Now().UnixNano()%10_000_000_000, 10)
	srtFilename := "subtitles-" + randomSuffix + ".srt"

	// Convert and write SRT
	err = writeSRT(srtFilename, draftContent.Tracks, textMap, jsonFilename) // Pass jsonFilename
	if err != nil {
		fmt.Println(err)
		return
	}

	fmt.Printf("Successfully converted subtitles from '%s' to %s\n", jsonFilename, srtFilename)
}
