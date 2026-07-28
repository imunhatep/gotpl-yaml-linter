# gotpl-linter — Go template YAML (Helm) linting & formatting

A small CLI that lints and formats Go-template YAML files, primarily Helm chart
templates. It re-indents template blocks based on **go-template control-structure
depth** (`if` / `range` / `with` / `define` … `end`) rather than YAML nesting, so
nested `{{- if }}` / `{{- end }}` blocks line up consistently at two spaces per level.

Built with Go 1.22.

## Install

```bash
go install github.com/imunhatep/gotpl-yaml-linter/cmd/gotpl-linter@latest

gotpl-linter --help
```

## Usage

The tool has two commands:

- **`lint`** — validate formatting; **writes nothing**. Exits non-zero if any file
  is not correctly formatted. Use this in CI.
- **`fmt`** — format files **in place**, rewriting any file that differs.

```
NAME:
   gotpl-linter - Go template YAML (Helm) formatting and linting tool

USAGE:
   gotpl-linter [global options] command [command options]

COMMANDS:
   fmt      format yaml tpl files in place
   lint     validate yaml gotpl formatting (no changes written)
   help, h  Shows a list of commands or help for one command

GLOBAL OPTIONS:
   --verbose value, --vv value  Log verbosity (default: 3) [$APP_DEBUG]
   --help, -h                   show help
   --version, -v                print the version
```

`--vv` sets log verbosity: `0` fatal, `1` error, `2` warn, `3` info (default),
`4` debug, `5`+ trace. It can also be set via the `APP_DEBUG` environment variable.

### Command options

Both `lint` and `fmt` take the same flags:

| Flag | Alias | Default | Description |
|------|-------|---------|-------------|
| `--path`   | `-p` | `./` | Directory to scan for template files |
| `--filter` | `-f` | `*`  | Glob pattern to match files (single directory level, non-recursive) |
| `--show`   | `-s` | `false` | Print the expected formatting to stdout |
| `--trim`   | `-t` | `false` | Rewrite non-`{{-` template openings to `{{-` so they can be re-indented |

> Note: `--filter` is a single-level glob matched against `<path>/<filter>`; it does
> not recurse into subdirectories.

### Re-indentation and rendered output

Re-indentation only changes the **leading whitespace** of template lines. That
whitespace is stripped at render time *only* when the line's opening action
left-trims (`{{-`). A line that opens with a plain `{{` has its leading whitespace
rendered literally into the output, so re-indenting it would shift the emitted YAML.

- **By default** the tool is output-safe: lines that do **not** left-trim are left
  exactly as they are (their block depth is still tracked, so surrounding `{{-` lines
  indent correctly). Lines that already use `{{-` are re-indented.
- **With `--trim` (`-t`)** the tool rewrites each non-`{{-` opening to `{{-` and then
  re-indents it. This normalises indentation everywhere but **can change the rendered
  output**, so review the diff before committing.

### Lint

Validate that files are correctly formatted. Nothing is written; a non-zero exit
code indicates at least one file is not formatted as expected.

```bash
gotpl-linter --vv 10 lint --path ./templates --filter '*.yaml'
```

### Format

Rewrite files in place to the expected formatting.

```bash
gotpl-linter --vv 10 fmt --path ./templates --filter '*.yaml'
```

Add `--show` to either command to print the expected output:

```bash
gotpl-linter lint -p ./templates -f '*.yaml' --show
```

## Examples

Formatting reindents template control lines to two spaces per block level. Plain
YAML lines are left untouched — only go-template lines are re-indented.

### Example 1

Input:

```gotemplate
{{- if or (eq .Values.controller.kind "Deployment") (eq .Values.controller.kind "Both") -}}
{{- include  "isControllerTagValid" . -}}
    {{- include "ingress-nginx.labels" . | nindent 4 }}
{{- end }}
```

Output:

```gotemplate
{{- if or (eq .Values.controller.kind "Deployment") (eq .Values.controller.kind "Both") -}}
  {{- include  "isControllerTagValid" . -}}
  {{- include "ingress-nginx.labels" . | nindent 4 }}
{{- end }}
```

### Example 2

Input:

```gotemplate
{{- if or (eq .Values.controller.kind "Deployment") (eq .Values.controller.kind "Both") -}}
{{- include  "isControllerTagValid" . -}}
apiVersion: apps/v1
kind: Deployment
metadata:
  labels:
    {{- include "ingress-nginx.labels" . | nindent 4 }}
    app.kubernetes.io/component: controller
    {{- with .Values.controller.labels }}
    {{- toYaml . | nindent 4 }}
    {{- end }}
  name: {{ include "ingress-nginx.controller.fullname" . }}
  namespace: {{ .Release.Namespace }}
  {{- if .Values.controller.annotations }}
  annotations: {{ toYaml .Values.controller.annotations | nindent 4 }}
  {{- end }}
{{- end }}
```

Output:

```gotemplate
{{- if or (eq .Values.controller.kind "Deployment") (eq .Values.controller.kind "Both") -}}
  {{- include  "isControllerTagValid" . -}}
apiVersion: apps/v1
kind: Deployment
metadata:
  labels:
  {{- include "ingress-nginx.labels" . | nindent 4 }}
    app.kubernetes.io/component: controller
  {{- with .Values.controller.labels }}
    {{- toYaml . | nindent 4 }}
  {{- end }}
  name: {{ include "ingress-nginx.controller.fullname" . }}
  namespace: {{ .Release.Namespace }}
  {{- if .Values.controller.annotations }}
  annotations: {{ toYaml .Values.controller.annotations | nindent 4 }}
  {{- end }}
{{- end }}
```

### Example 3

Input:

```gotemplate
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
{{- end }}
```

Output:

```gotemplate
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
{{- end }}
```

## Build from source

```bash
make          # build binary into dist/
make test     # gofmt check + go vet + golint + go test
make xb       # cross-build for linux/darwin/windows
```
