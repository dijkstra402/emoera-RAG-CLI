package cmd

import (
	"fmt"
	"io"
	"os"
	"strings"

	"github.com/dijkstra402/emoera-RAG-CLI/internal/api"
	"golang.org/x/term"
)

const (
	ansiReset  = "\x1b[0m"
	ansiBold   = "\x1b[1m"
	ansiDim    = "\x1b[2m"
	ansiAccent = "\x1b[38;5;203m"
	ansiGreen  = "\x1b[38;5;71m"
)

type askPresenter struct {
	writer io.Writer
	color  bool
	opened bool
}

func newAskPresenter(writer io.Writer, noColor bool) *askPresenter {
	return &askPresenter{writer: writer, color: supportsColor(writer, noColor)}
}

func (presenter *askPresenter) stage(stage string) {
	if !presenter.opened {
		fmt.Fprintf(presenter.writer, "%s\n", presenter.paint(ansiBold+ansiAccent, "EMOERA  知识库问答"))
		presenter.opened = true
	}
	label, known := askStageLabels[stage]
	if !known {
		label = stage
	}
	marker := presenter.paint(ansiGreen, "◆")
	fmt.Fprintf(presenter.writer, "  %s %s\n", marker, presenter.paint(ansiDim, label))
}

func (presenter *askPresenter) answerHeading() {
	if !presenter.opened {
		fmt.Fprintf(presenter.writer, "%s\n", presenter.paint(ansiBold+ansiAccent, "EMOERA  知识库问答"))
		presenter.opened = true
	}
	fmt.Fprintf(presenter.writer, "\n%s\n", presenter.paint(ansiBold, "回答"))
}

func (presenter *askPresenter) summary(result api.ChatCompletionResult) {
	fmt.Fprintln(presenter.writer, presenter.paint(ansiDim, "────────────────────"))

	if names := referenceNames(result.References); len(names) > 0 {
		fmt.Fprintf(presenter.writer, "%s  %s\n",
			presenter.paint(ansiBold, "来源"), strings.Join(names, " · "))
	}
	if result.SessionUUID != "" {
		fmt.Fprintf(presenter.writer, "%s  %s\n",
			presenter.paint(ansiBold, "会话"), presenter.paint(ansiDim, result.SessionUUID))
	}
	if len(result.Suggestions) > 0 {
		fmt.Fprintf(presenter.writer, "\n%s\n", presenter.paint(ansiBold, "可以继续问"))
		for index, suggestion := range result.Suggestions {
			fmt.Fprintf(presenter.writer, "  %d. %s\n", index+1, suggestion)
		}
	}
}

func (presenter *askPresenter) paint(style, value string) string {
	if !presenter.color {
		return value
	}
	return style + value + ansiReset
}

var askStageLabels = map[string]string{
	"rewriting":  "理解并改写问题",
	"retrieving": "检索知识库",
	"reranking":  "重排高相关内容",
	"generating": "组织回答",
}

func referenceNames(references []map[string]any) []string {
	seen := make(map[string]struct{}, len(references))
	names := make([]string, 0, len(references))
	for _, reference := range references {
		name, _ := reference["fileName"].(string)
		name = strings.TrimSpace(name)
		if name == "" {
			continue
		}
		if _, exists := seen[name]; exists {
			continue
		}
		seen[name] = struct{}{}
		names = append(names, name)
		if len(names) == 4 {
			break
		}
	}
	return names
}

func supportsColor(writer io.Writer, noColor bool) bool {
	if noColor || os.Getenv("NO_COLOR") != "" || os.Getenv("TERM") == "dumb" {
		return false
	}
	type fileDescriptor interface{ Fd() uintptr }
	terminal, ok := writer.(fileDescriptor)
	return ok && term.IsTerminal(int(terminal.Fd()))
}
