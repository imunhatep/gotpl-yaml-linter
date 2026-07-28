package app

import (
	"fmt"
	"os"
	"regexp"
	"strings"

	"github.com/rs/zerolog/log"
)

const (
	controlStartPattern    = `{{-?\s*(if|range|with|define|block)\s`
	controlContinuePattern = `{{-?\s*(else)\s`
	controlEndPattern      = `{{-?\s*end\s*-?}}`
	// nonControlPattern matches a line that opens with a go-template action
	// ({{ ... }}) but is not itself a control keyword. Openers, continuations,
	// closers, comments and variable assignments have dedicated patterns and
	// are classified earlier in the chain, so any remaining {{-led line (a bare
	// value, a helper call such as include/toYaml/tpl/printf, etc.) is treated
	// as a non-structural line indented to the current block level.
	nonControlPattern      = `{{`
	templateCommentPattern = `{{-?\s*/\*`
	// variableAssignmentPattern matches go-template variable declarations like
	// {{ $var := ... }} or {{- $var := ... -}} and should be treated as a
	// non-structural line (indented to the current block level).
	variableAssignmentPattern = `{{-?\s*\$[A-Za-z0-9_.-]+\s*\:?=`
)

var (
	// Anchored variants: must start at beginning of the trimmed line
	controlStructureStart    = regexp.MustCompile(`^` + controlStartPattern)
	controlStructureContinue = regexp.MustCompile(`^` + controlContinuePattern)
	controlStructureEnd      = regexp.MustCompile(`^` + controlEndPattern)
	nonControlStructure      = regexp.MustCompile(`^` + nonControlPattern)

	// Unanchored variants: match tokens anywhere in the line. Used to detect
	// inline templates that both open and close on the same line.
	containsControlStructureStart = regexp.MustCompile(controlStartPattern)
	containsControlStructureEnd   = regexp.MustCompile(controlEndPattern)

	// templateComment matches helm/go-template comments that start with {{/* or {{- /*
	// Example: {{/* some comment */}} or {{-/* comment */ -}}
	templateComment    = regexp.MustCompile(`^` + templateCommentPattern)
	variableAssignment = regexp.MustCompile(`^` + variableAssignmentPattern)
)

// FormatYamlTpl formats a yaml template string.
//
// Re-indentation only alters the leading whitespace of template lines. That
// whitespace is stripped at render time only when the line's opening action
// left-trims ({{-), so by default a line that does not left-trim is emitted
// unchanged to avoid shifting the rendered output. When forceTrim is true such
// a line's opening {{ is rewritten to {{- so it can be safely re-indented.
// Block depth is tracked from the control structures regardless of trim markers.
func FormatYamlTpl(yamlTpl string, forceTrim bool) (string, error) {
	lines := strings.Split(yamlTpl, "\n")

	indentLevel := 0
	var formattedLines []string
	for i, line := range lines {
		trimmed := strings.TrimSpace(strings.ReplaceAll(line, "\t", "\n"))
		// If a line contains both a start and an end control structure (inline template),
		// treat it as a single-line template and do not modify indentLevel.
		if containsControlStructureStart.MatchString(trimmed) && containsControlStructureEnd.MatchString(trimmed) {
			formattedLines = append(formattedLines, emitTemplateLine(line, trimmed, indentLevel, forceTrim))
			continue
		}

		if isStartControlStructure(trimmed) {
			formattedLines = append(formattedLines, emitTemplateLine(line, trimmed, indentLevel, forceTrim))
			indentLevel++
		} else if isContinueControlStructure(trimmed) {
			// else / else-if stay visually at the parent block level. Guard
			// against malformed input (an else with no opening structure) that
			// would otherwise pass a negative level to strings.Repeat and panic.
			emitLevel := indentLevel - 1
			if emitLevel < 0 {
				emitLevel = 0
				log.Warn().Msgf("Seems 'else' has no opening structure. Invalid gotpl structure at line ~%d: %s", i, line)
			}

			formattedLines = append(formattedLines, emitTemplateLine(line, trimmed, emitLevel, forceTrim))
		} else if isEndControlStructure(trimmed) {
			// End control structures are indented according to their current block level.
			// Checked before the non-control catch-all below, which also matches {{ end }}.
			indentLevel--

			if indentLevel < 0 {
				indentLevel = 0
				log.Warn().Msgf("Seems closing structure has no opening. Invalid gotpl structure at line ~%d: %s", i, line)
			}

			formattedLines = append(formattedLines, emitTemplateLine(line, trimmed, indentLevel, forceTrim))
		} else if isTemplateComment(trimmed) {
			// Template comments should be ignored for control-structure processing
			// and indented according to the current block level.
			formattedLines = append(formattedLines, emitTemplateLine(line, trimmed, indentLevel, forceTrim))
		} else if isVariableAssignment(trimmed) {
			// Variable assignment lines ({{ $x := ... }}) are not control structures
			// but should be indented to the current block level.
			formattedLines = append(formattedLines, emitTemplateLine(line, trimmed, indentLevel, forceTrim))
		} else if isNonControlStructure(trimmed) {
			// Any other standalone template line is indented to the current block level
			formattedLines = append(formattedLines, emitTemplateLine(line, trimmed, indentLevel, forceTrim))
		} else {
			// Regular lines that are not control structures or non-control structures are treated as text
			formattedLines = append(formattedLines, line)
		}
	}

	return strings.Join(formattedLines, "\n"), nil
}

// FormatYamlTplFile formats a yaml file
func FormatYamlTplFile(file string, format, output, forceTrim bool) (bool, error) {
	original, err := os.ReadFile(file)
	if err != nil {
		return false, err
	}

	data, err := FormatYamlTpl(string(original), forceTrim)
	if err != nil {
		return false, err
	}

	// output expected file formatting
	if output {
		fmt.Printf("\nexpected yaml [%s] tpl formtting:\n%s\n\n", file, data)
	}

	// yaml are invalid
	if string(original) == data {
		log.Info().Str("file", file).Msgf("yaml template is valid")
		return true, nil
	}

	// validate, do not change files
	if !format {
		log.Error().Str("file", file).Msgf("error! yaml is invalid")
		return false, nil
	}

	// Write the new content to the file, overwriting existing content
	if err = os.WriteFile(file, []byte(data), 0644); err != nil {
		return false, err
	}

	log.Info().Str("file", file).Msg("linted")

	return true, nil
}

func formatLine(line string, indentLevel int) string {
	// Remove leading spaces to reset indentation
	trimmedLine := strings.TrimLeft(line, " ")
	return strings.Repeat("  ", indentLevel) + trimmedLine
}

// emitTemplateLine renders a standalone template line at the given block level.
// A line that left-trims ({{-) has its leading whitespace stripped at render
// time, so it is re-indented. A line that does not left-trim would have its
// leading whitespace rendered literally; by default it is emitted unchanged to
// preserve the rendered output. When forceTrim is set, its opening {{ is
// rewritten to {{- so it can be re-indented safely.
func emitTemplateLine(line, trimmed string, indentLevel int, forceTrim bool) string {
	if hasLeftTrim(trimmed) {
		return formatLine(line, indentLevel)
	}

	if forceTrim {
		return formatLine(addLeftTrim(line), indentLevel)
	}

	return line
}

// hasLeftTrim reports whether the line's leading template action uses the
// left-trim marker ({{-).
func hasLeftTrim(trimmed string) bool {
	return strings.HasPrefix(trimmed, "{{-")
}

// addLeftTrim rewrites the first (opening) {{ on the line to a left-trimming
// {{- marker, normalising the whitespace after it to a single space so the
// result is a valid trim marker (e.g. "  {{ x }}" -> "  {{- x }}").
func addLeftTrim(line string) string {
	before, after, found := strings.Cut(line, "{{")
	if !found {
		return line
	}

	return before + "{{- " + strings.TrimLeft(after, " \t")
}

func isStartControlStructure(line string) bool {
	lineWithoutLeadingSpaces := strings.TrimSpace(line)
	return controlStructureStart.MatchString(lineWithoutLeadingSpaces)
}

func isContinueControlStructure(line string) bool {
	lineWithoutLeadingSpaces := strings.TrimSpace(line)
	return controlStructureContinue.MatchString(lineWithoutLeadingSpaces)
}

func isEndControlStructure(line string) bool {
	lineWithoutLeadingSpaces := strings.TrimSpace(line)
	return controlStructureEnd.MatchString(lineWithoutLeadingSpaces)
}

func isNonControlStructure(line string) bool {
	lineWithoutLeadingSpaces := strings.TrimSpace(line)
	return nonControlStructure.MatchString(lineWithoutLeadingSpaces)
}

func isTemplateComment(line string) bool {
	lineWithoutLeadingSpaces := strings.TrimSpace(line)
	return templateComment.MatchString(lineWithoutLeadingSpaces)
}

func isVariableAssignment(line string) bool {
	lineWithoutLeadingSpaces := strings.TrimSpace(line)
	return variableAssignment.MatchString(lineWithoutLeadingSpaces)
}
