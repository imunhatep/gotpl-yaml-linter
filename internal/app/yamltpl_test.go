package app

import (
	"testing"
)

func TestFormatLine(t *testing.T) {
	tests := []struct {
		line        string
		indentLevel int
		want        string
	}{
		{"test line", 1, "  test line"},
		{"  test line", 2, "    test line"},
		{"test line", 0, "test line"},
	}

	for _, tt := range tests {
		got := formatLine(tt.line, tt.indentLevel)
		if got != tt.want {
			t.Errorf("formatLine(%q, %d) = %q, want %q", tt.line, tt.indentLevel, got, tt.want)
		}
	}
}

func TestIsStartControlStructure(t *testing.T) {
	tests := []struct {
		line string
		want bool
	}{
		{"{{ if .Condition }}", true},
		{"{{- if .Condition }}", true},
		{"{{ range .Items }}", true},
		{"{{ with .Context }}", true},
		{"Not a control structure", false},
	}

	for _, tt := range tests {
		got := isStartControlStructure(tt.line)
		if got != tt.want {
			t.Errorf("isStartControlStructure(%q) = %t, want %t", tt.line, got, tt.want)
		}
	}
}

func TestIsEndControlStructure(t *testing.T) {
	tests := []struct {
		line string
		want bool
	}{
		{"{{ end }}", true},
		{"{{- end -}}", true},
		{"Not an end control structure", false},
	}

	for _, tt := range tests {
		got := isEndControlStructure(tt.line)
		if got != tt.want {
			t.Errorf("isEndControlStructure(%q) = %t, want %t", tt.line, got, tt.want)
		}
	}
}

func TestIsNonControlStructure(t *testing.T) {
	tests := []struct {
		line string
		want bool
	}{
		{"{{ include \"template\" . }}", true},
		{"{{ toYaml . }}", true},
		{"{{ nindent 2 . }}", true},
		{"Not a non-control structure", false},
	}

	for _, tt := range tests {
		got := isNonControlStructure(tt.line)
		if got != tt.want {
			t.Errorf("isNonControlStructure(%q) = %t, want %t", tt.line, got, tt.want)
		}
	}
}

func TestIsTemplateComment(t *testing.T) {
	tests := []struct {
		line string
		want bool
	}{
		{"{{/* this is a comment */}}", true},
		{"{{-/* comment */-}}", true},
		{"Not a comment", false},
	}

	for _, tt := range tests {
		got := isTemplateComment(tt.line)
		if got != tt.want {
			t.Errorf("isTemplateComment(%q) = %t, want %t", tt.line, got, tt.want)
		}
	}
}

func TestFormatYamlTpl(t *testing.T) {
	tests := []struct {
		name    string
		yamlTpl string
		want    string
	}{
		{
			name: "No indentation needed",
			yamlTpl: `apiVersion: v1
{{ if .Condition }}
kind: Pod
{{ end }}`,
			want: `apiVersion: v1
{{ if .Condition }}
kind: Pod
{{ end }}`,
		},
		{
			name: "Variable assignment indented",
			yamlTpl: `apiVersion: v1
{{- if .Condition }}
{{ $commonFilePath := printf "files/all/*" -}}
kind: Pod
{{- end }}`,
			want: `apiVersion: v1
{{- if .Condition }}
  {{ $commonFilePath := printf "files/all/*" -}}
kind: Pod
{{- end }}`,
		},
		{
			name: "Inline control structure on single line",
			yamlTpl: `apiVersion: v1
{{- if .Values.deploymentId -}}-{{- .Values.deploymentId | toString | replace "." "-" -}}{{- end -}}
kind: Pod`,
			want: `apiVersion: v1
{{- if .Values.deploymentId -}}-{{- .Values.deploymentId | toString | replace "." "-" -}}{{- end -}}
kind: Pod`,
		},
		{
			name: "Nested structures",
			yamlTpl: `apiVersion: v1
{{ if .Condition }}
kind: Pod
{{ with .Spec }}
spec:
  containers:
  - name: my-container
{{ end }}
{{ end }}`,
			want: `apiVersion: v1
{{ if .Condition }}
kind: Pod
  {{ with .Spec }}
spec:
  containers:
  - name: my-container
  {{ end }}
{{ end }}`,
		},
		{
			name: "Mixed control and non-control structures",
			yamlTpl: `apiVersion: v1
{{- if .Condition }}
kind: Pod
{{- include "myTemplate" . | nindent 2 }}
{{- end }}`,
			want: `apiVersion: v1
{{- if .Condition }}
kind: Pod
  {{- include "myTemplate" . | nindent 2 }}
{{- end }}`,
		},
		{
			name: "Simple if structure",
			yamlTpl: `apiVersion: v1
{{- if or (eq .Values.controller.kind "Deployment") (eq .Values.controller.kind "Both") -}}
{{- include  "isControllerTagValid" . -}}
    {{- include "ingress-nginx.labels" . | nindent 4 }}
{{- end }}`,
			want: `apiVersion: v1
{{- if or (eq .Values.controller.kind "Deployment") (eq .Values.controller.kind "Both") -}}
  {{- include  "isControllerTagValid" . -}}
  {{- include "ingress-nginx.labels" . | nindent 4 }}
{{- end }}`,
		},
		{
			name: "Template comments ignored",
			yamlTpl: `apiVersion: v1
{{- if .Condition }}
{{/* helm comment */}}
kind: Pod
{{- end }}`,
			want: `apiVersion: v1
{{- if .Condition }}
  {{/* helm comment */}}
kind: Pod
{{- end }}`,
		},
		{
			name: "Simple if structure",
			yamlTpl: `apiVersion: v1
{{- if or (eq .Values.controller.kind "Deployment") (eq .Values.controller.kind "Both") -}}
{{- include  "isControllerTagValid" . -}}
    {{- include "ingress-nginx.labels" . | nindent 4 }}
    {{- with .Values.controller.labels }}
    {{- toYaml . | nindent 4 }}
    {{- end }}
  {{- if .Values.controller.annotations }}
    {{- with .Values.controller.labels }}
    {{- toYaml . | nindent 8 }}
    {{- end }}
  {{- end }}
      {{- include "ingress-nginx.selectorLabels" . | nindent 6 }}
  {{- if not .Values.controller.autoscaling.enabled }}
  {{- end }}
{{- end }}`,
			want: `apiVersion: v1
{{- if or (eq .Values.controller.kind "Deployment") (eq .Values.controller.kind "Both") -}}
  {{- include  "isControllerTagValid" . -}}
  {{- include "ingress-nginx.labels" . | nindent 4 }}
  {{- with .Values.controller.labels }}
    {{- toYaml . | nindent 4 }}
  {{- end }}
  {{- if .Values.controller.annotations }}
    {{- with .Values.controller.labels }}
      {{- toYaml . | nindent 8 }}
    {{- end }}
  {{- end }}
  {{- include "ingress-nginx.selectorLabels" . | nindent 6 }}
  {{- if not .Values.controller.autoscaling.enabled }}
  {{- end }}
{{- end }}`,
		},
	}

	for _, tt := range tests {
		got, err := FormatYamlTpl(tt.yamlTpl)
		if err != nil {
			t.Errorf("%s: unexpected error: %v", tt.name, err)
			continue
		}

		if got != tt.want {
			t.Errorf("%s: got:\n%s\nwant:\n%s", tt.name, got, tt.want)
		}
	}
}
