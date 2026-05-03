package main

import (
	"fmt"
	"path/filepath"
	"regexp"
	"strings"
	"time"

	midipkg "github.com/Andrea-Cavallo/cadenza/internal/midi"
)

const editedTicksPerStep = 120

var safeFilePartRE = regexp.MustCompile(`[^a-zA-Z0-9_-]+`)

func (a *AppService) ExportEditedPreview(req ExportEditedRequest) (ExportEditedResult, error) {
	req = normalizeExportEditedRequest(req)
	if len(req.Preview.Patterns) == 0 {
		return ExportEditedResult{}, fmt.Errorf("no edited preview to export")
	}

	a.ensureLogger()
	a.emitProgressStep("write", "exporting edited piano roll")

	writer := midipkg.NewWriter(float64(req.BPM))
	dir := filepath.Join(a.defaultOutputDir(), "edited")
	stamp := time.Now().Format("20060102_150405")

	files := make([]string, 0, len(req.Preview.Patterns))
	for _, track := range req.Preview.Patterns {
		events := editedTrackEvents(track)
		if len(events) == 0 {
			continue
		}
		name := fmt.Sprintf("edited_%s_%s.mid", safeFilePart(track.PatternType), stamp)
		path := filepath.Join(dir, name)
		if err := writer.WriteFile(path, events); err != nil {
			return ExportEditedResult{}, fmt.Errorf("%s export: %w", track.PatternType, err)
		}
		files = append(files, path)
	}

	if len(files) == 0 {
		return ExportEditedResult{}, fmt.Errorf("edited preview contains no active notes")
	}
	a.emitProgressStep("write", fmt.Sprintf("edited export done - %d files written", len(files)))
	return ExportEditedResult{Files: files}, nil
}

func normalizeExportEditedRequest(req ExportEditedRequest) ExportEditedRequest {
	if req.BPM == 0 {
		req.BPM = 122
	}
	if req.Preview.StepsPerBar == 0 {
		req.Preview.StepsPerBar = 16
	}
	if req.Preview.Bars == 0 {
		req.Preview.Bars = 16
	}
	return req
}

func editedTrackEvents(track TrackPreview) []midipkg.MIDIEvent {
	events := make([]midipkg.MIDIEvent, 0, len(track.Steps)*2)
	for _, step := range track.Steps {
		if !step.Active || step.MIDI <= 0 {
			continue
		}
		start := int64(max(step.Step, 0) * editedTicksPerStep)
		duration := max(step.DurationSteps, 1)
		end := start + int64(duration*editedTicksPerStep)
		note := uint8(clampInt(step.MIDI, 0, 127))
		velocity := uint8(clampInt(step.Velocity, 1, 120))

		events = append(events,
			midipkg.MIDIEvent{
				Type:     midipkg.NoteOn,
				Tick:     start,
				Channel:  0,
				Note:     note,
				Velocity: velocity,
			},
			midipkg.MIDIEvent{
				Type:    midipkg.NoteOff,
				Tick:    end,
				Channel: 0,
				Note:    note,
			},
		)
	}
	return events
}

func safeFilePart(value string) string {
	value = strings.TrimSpace(strings.ToLower(value))
	if value == "" {
		return "track"
	}
	return strings.Trim(safeFilePartRE.ReplaceAllString(value, "_"), "_")
}

func clampInt(value, minValue, maxValue int) int {
	if value < minValue {
		return minValue
	}
	if value > maxValue {
		return maxValue
	}
	return value
}
