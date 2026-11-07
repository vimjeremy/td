package parse

import (
	"fmt"
	"strings"
	"time"

	"github.com/jeremy/td/internal/model"
)

// Parsed is the result of parsing quick-add text such as
// "Call dentist +personal @phone due:tomorrow".
type Parsed struct {
	Title      string
	Project    string
	HasProject bool // +name was given (+none clears the project)
	Tags       []string
	Due        *time.Time
	ClearDue   bool // due:none was given
	Dir        string
	HasDir     bool // dir: was given (dir:none clears it)
	Note       string
	HasNote    bool // note: was given (note:none clears it)
}

// Input parses quick-add tokens. Arguments are split on whitespace so a single
// quoted string works as well as separate words.
func Input(args []string, now time.Time) (Parsed, error) {
	var p Parsed
	var words, noteWords []string
	inNote := false
	for _, arg := range args {
		for _, tok := range strings.Fields(arg) {
			if inNote { // note: takes the rest of the line
				noteWords = append(noteWords, tok)
				continue
			}
			switch {
			case strings.HasPrefix(tok, "note:"):
				inNote, p.HasNote = true, true
				if rest := tok[len("note:"):]; rest != "" {
					noteWords = append(noteWords, rest)
				}
			case len(tok) > 1 && tok[0] == '+':
				p.HasProject = true
				p.Project = tok[1:]
				if p.Project == "none" {
					p.Project = ""
				}
			case len(tok) > 1 && tok[0] == '@':
				p.Tags = appendUnique(p.Tags, tok[1:])
			case strings.HasPrefix(tok, "dir:"):
				p.HasDir = true
				p.Dir = ""
				if v := tok[len("dir:"):]; v != "" && v != "none" {
					d, err := ExpandDir(v)
					if err != nil {
						return p, err
					}
					p.Dir = d
				}
			case strings.HasPrefix(tok, "due:"):
				v := tok[len("due:"):]
				if v == "" || v == "none" {
					p.ClearDue = true
					p.Due = nil
					continue
				}
				d, err := Date(v, now)
				if err != nil {
					return p, err
				}
				p.Due, p.ClearDue = &d, false
			default:
				words = append(words, tok)
			}
		}
	}
	p.Title = strings.Join(words, " ")
	if p.Note = strings.Join(noteWords, " "); p.Note == "none" {
		p.Note = ""
	}
	return p, nil
}

func appendUnique(list []string, s string) []string {
	for _, x := range list {
		if x == s {
			return list
		}
	}
	return append(list, s)
}

// ToTask builds a brand-new task from everything that was parsed.
func (p Parsed) ToTask() (model.Task, error) {
	if p.Title == "" {
		return model.Task{}, fmt.Errorf("a task needs a title")
	}
	return model.Task{
		Title:   p.Title,
		Project: p.Project,
		Dir:     p.Dir,
		Notes:   p.Note,
		Tags:    p.Tags,
		Due:     p.Due,
	}, nil
}

// Apply merges only the fields that were explicitly given into t. Tags, when
// given, replace the existing tags.
func (p Parsed) Apply(t *model.Task) {
	if p.Title != "" {
		t.Title = p.Title
	}
	if p.HasProject {
		t.Project = p.Project
	}
	if p.HasDir {
		t.Dir = p.Dir
	}
	if p.HasNote {
		t.Notes = p.Note
	}
	if len(p.Tags) > 0 {
		t.Tags = p.Tags
	}
	if p.Due != nil {
		t.Due = p.Due
	}
	if p.ClearDue {
		t.Due = nil
	}
}
