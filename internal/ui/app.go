// Package ui implements the full-screen Bubble Tea interface.
package ui

import (
	"fmt"
	"os"
	"os/exec"
	"strconv"
	"strings"
	"time"

	"github.com/charmbracelet/bubbles/textinput"
	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
	"github.com/jeremy/td/internal/model"
	"github.com/jeremy/td/internal/parse"
	"github.com/jeremy/td/internal/render"
	"github.com/jeremy/td/internal/store"
)

const sidebarWidth = 22

type mode int

const (
	modeNormal mode = iota
	modeInput
	modeConfirm
)

type inputKind int

const (
	inAdd inputKind = iota
	inEdit
	inFilter
)

type viewKind int

const (
	vUpcoming viewKind = iota
	vDone
	vProject
	vTag
)

// view is one sidebar entry and the filter it selects.
type view struct {
	kind   viewKind
	label  string
	header bool // non-selectable section heading
	filter func(now time.Time) model.Filter
}

// Model is the Bubble Tea model for the whole app.
type Model struct {
	st        *store.Store
	views     []view
	viewIdx   int
	focusList bool
	tasks     []model.Task // after the text filter
	cursor    int
	mode      mode
	kind      inputKind
	input     textinput.Model
	form      [6]textinput.Model // add/edit form, see the f* constants
	formIdx   int
	origNotes string         // notes of the task being edited, to keep multi-line text intact
	expanded  map[int64]bool // tasks whose details are shown
	text      string         // active "/" filter
	status    string
	showHelp  bool
	w, h      int
}

// Run starts the interface and blocks until the user quits.
func Run(st *store.Store) error {
	in := textinput.New()
	in.Prompt = "> "
	m := &Model{st: st, input: in, focusList: true, expanded: map[int64]bool{}}
	m.reload()
	_, err := tea.NewProgram(m, tea.WithAltScreen()).Run()
	return err
}

func (m *Model) Init() tea.Cmd { return nil }

// buildViews rebuilds the sidebar from the current projects and tags.
func (m *Model) buildViews() {
	vs := []view{
		{label: "All", header: true},
		{kind: vUpcoming, label: "Upcoming", filter: func(n time.Time) model.Filter { return model.Filter{} }}, // everything still to be done
		{kind: vDone, label: "Done", filter: func(time.Time) model.Filter { return model.Filter{DoneOnly: true} }},
	}
	var projects []string
	if ps, _ := m.st.Projects(); len(ps) > 0 {
		for _, p := range ps {
			if p.Total > 0 { // keep a project until its last task is deleted
				projects = append(projects, p.Name)
			}
		}
	}
	if len(projects) > 0 {
		vs = append(vs, view{label: "Projects", header: true})
		for _, name := range projects {
			vs = append(vs, view{kind: vProject, label: name, filter: func(time.Time) model.Filter { return model.Filter{Project: name, All: true} }})
		}
	}
	var tags []string
	if ts, _ := m.st.Tags(); len(ts) > 0 {
		for _, t := range ts {
			if t.Total > 0 { // keep a tag until its last task is deleted
				tags = append(tags, t.Name)
			}
		}
	}
	if len(tags) > 0 {
		vs = append(vs, view{label: "Tags", header: true})
		for _, name := range tags {
			vs = append(vs, view{kind: vTag, label: name, filter: func(time.Time) model.Filter { return model.Filter{Tag: name, All: true} }})
		}
	}
	var prev view
	if m.viewIdx < len(m.views) {
		prev = m.views[m.viewIdx]
	}
	m.views = vs
	m.viewIdx = 0
	for i, v := range vs { // default: the first entry that is not a heading
		if !v.header {
			m.viewIdx = i
			break
		}
	}
	for i, v := range vs {
		if !v.header && v.kind == prev.kind && v.label == prev.label {
			m.viewIdx = i
		}
	}
}

// reload refreshes the sidebar and the task list from the database.
func (m *Model) reload() {
	m.buildViews()
	tasks, err := m.st.List(m.views[m.viewIdx].filter(time.Now()))
	if err != nil {
		m.status = "error: " + err.Error()
		return
	}
	if m.text != "" {
		needle := strings.ToLower(m.text)
		kept := tasks[:0]
		for _, t := range tasks {
			if strings.Contains(strings.ToLower(searchText(t)), needle) {
				kept = append(kept, t)
			}
		}
		tasks = kept
	}
	m.tasks = tasks
	m.cursor = max(0, min(m.cursor, len(tasks)-1))
}

func (m *Model) current() (model.Task, bool) {
	if m.cursor < 0 || m.cursor >= len(m.tasks) {
		return model.Task{}, false
	}
	return m.tasks[m.cursor], true
}

func (m *Model) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	switch msg := msg.(type) {
	case tea.WindowSizeMsg:
		m.w, m.h = msg.Width, msg.Height
		return m, nil
	case shellDoneMsg:
		m.reload() // the shell may have run td commands
		if msg.err != nil {
			m.status = "shell: " + msg.err.Error()
		}
		return m, nil
	case tea.KeyMsg:
		switch m.mode {
		case modeInput:
			return m.updateInput(msg)
		case modeConfirm:
			return m.updateConfirm(msg)
		}
		return m.updateNormal(msg)
	}
	return m, nil
}

// moveSection jumps the sidebar between its sections: the main tabs, PROJECTS and TAGS.
func (m *Model) moveSection(dir int) {
	var starts []int // first selectable entry of each section
	for i, v := range m.views {
		if !v.header && (i == 0 || m.views[i-1].header) {
			starts = append(starts, i)
		}
	}
	cur := 0
	for k, st := range starts {
		if st <= m.viewIdx {
			cur = k
		}
	}
	m.viewIdx, m.cursor = starts[(cur+dir+len(starts))%len(starts)], 0
	m.reload()
}

// moveGroup jumps the cursor to the first task of the next (dir 1) or previous
// (dir -1) group in the list: Overdue, Today, Next 7 days, Later, No date, Done.
func (m *Model) moveGroup(dir int) {
	now := time.Now()
	var starts []int // index of the first task of each group shown
	for i, t := range m.tasks {
		if i == 0 || bucket(t, now) != bucket(m.tasks[i-1], now) {
			starts = append(starts, i)
		}
	}
	if len(starts) < 2 {
		return
	}
	cur := 0
	for k, st := range starts {
		if st <= m.cursor {
			cur = k
		}
	}
	m.cursor = starts[(cur+dir+len(starts))%len(starts)]
}

func (m *Model) moveView(dir int) {
	for i := m.viewIdx + dir; i >= 0 && i < len(m.views); i += dir {
		if !m.views[i].header {
			m.viewIdx, m.cursor = i, 0
			m.reload()
			return
		}
	}
}

func (m *Model) updateNormal(msg tea.KeyMsg) (tea.Model, tea.Cmd) {
	key := msg.String()
	m.status = ""
	if m.showHelp {
		m.showHelp = false
		if key != "q" && key != "ctrl+c" {
			return m, nil
		}
	}
	switch key {
	case "q", "ctrl+c":
		return m, tea.Quit
	case "?":
		m.showHelp = true
	case "esc":
		if m.text != "" {
			m.text = ""
			m.reload()
		}
	case "/":
		m.startInput(inFilter, m.text)
		return m, textinput.Blink
	case "a":
		var v formValues
		switch cur := m.views[m.viewIdx]; cur.kind { // prefill from the tab you are in
		case vTag:
			v.tag = cur.label
		case vProject:
			v.project = cur.label
		}
		m.startForm(inAdd, v)
		return m, textinput.Blink
	}

	if !m.focusList {
		switch key {
		case "j", "down":
			m.moveView(1)
		case "k", "up":
			m.moveView(-1)
		case "enter", "l", "right":
			m.focusList = true
		case "tab":
			m.moveSection(1)
		case "shift+tab":
			m.moveSection(-1)
		}
		return m, nil
	}

	t, ok := m.current()
	switch key {
	case "j", "down":
		m.cursor = max(0, min(m.cursor+1, len(m.tasks)-1))
	case "k", "up":
		m.cursor = max(0, m.cursor-1)
	case "g", "home":
		m.cursor = 0
	case "G", "end":
		m.cursor = max(0, len(m.tasks)-1)
	case "h", "left":
		m.focusList = false
	case "tab":
		m.moveGroup(1)
	case "shift+tab":
		m.moveGroup(-1)
	case " ", "x":
		if ok {
			m.check(m.st.SetDone(t.ID, !t.Done())) // toggles: reopens in the Done tab
		}
	case "enter":
		if ok {
			m.expanded[t.ID] = !m.expanded[t.ID]
		}
	case "e":
		if ok {
			date := ""
			if t.Due != nil {
				date = t.Due.Format("2006-01-02")
			}
			m.startForm(inEdit, formValues{title: t.Title, project: t.Project, tag: strings.Join(t.Tags, " "), date: date, dir: t.Dir, notes: t.Notes})
			return m, textinput.Blink
		}
	case "d":
		if ok {
			m.mode = modeConfirm
		}
	case "c":
		if ok {
			return m, m.openShell(t)
		}
	}
	return m, nil
}

type shellDoneMsg struct{ err error }

// openShell suspends the UI and runs $SHELL inside the task's directory; the UI
// comes back when the shell exits. (A program cannot change its parent shell's
// directory, so a subshell is the way to "cd" from here.)
func (m *Model) openShell(t model.Task) tea.Cmd {
	if t.Dir == "" {
		m.status = "this task has no directory"
		return nil
	}
	if fi, err := os.Stat(t.Dir); err != nil || !fi.IsDir() {
		m.status = "directory not found: " + t.Dir
		return nil
	}
	shell := os.Getenv("SHELL")
	if shell == "" {
		shell = "/bin/sh"
	}
	c := exec.Command(shell)
	c.Dir = t.Dir
	c.Env = append(os.Environ(), "TD_TASK_ID="+strconv.FormatInt(t.ID, 10))
	return tea.ExecProcess(c, func(err error) tea.Msg { return shellDoneMsg{err} })
}

// check reports a store error in the status line and reloads the list.
func (m *Model) check(err error) {
	if err != nil {
		m.status = "error: " + err.Error()
		return
	}
	m.reload()
}

func (m *Model) startInput(kind inputKind, value string) {
	m.mode, m.kind = modeInput, kind
	m.input.SetValue(value)
	m.input.CursorEnd()
	m.input.Focus()
	switch kind {
	case inAdd:
		m.input.Prompt = "add> "
	case inEdit:
		m.input.Prompt = "edit> "
	default:
		m.input.Prompt = "filter> "
	}
}

// Add/edit form fields, in Tab order (top to bottom, left to right).
const (
	fTitle = iota
	fProject
	fTag
	fDate
	fDir
	fNotes
)

type formValues struct{ title, project, tag, date, dir, notes string }

// flatten shows multi-line notes on the single-line notes field.
func flatten(s string) string { return strings.Join(strings.Fields(s), " ") }

func (m *Model) startForm(kind inputKind, v formValues) {
	m.mode, m.kind, m.status = modeInput, kind, ""
	wide := max(20, m.w-8)
	for i, f := range []struct {
		prompt, hint string
		width        int
	}{
		fTitle:   {"title ", "what needs doing?", wide},
		fProject: {"project ", "work", 14},
		fTag:     {"tag ", "errand, home", 18},
		fDate:    {"date ", "fri, +3d", 10},
		fDir:     {"dir ", "/path/the/task/relates/to (optional)", wide},
		fNotes:   {"notes ", "extra details (shown when you press enter on a task)", wide},
	} {
		in := textinput.New()
		in.Prompt, in.Placeholder, in.Width = f.prompt, f.hint, f.width
		m.form[i] = in
	}
	m.form[fTitle].SetValue(v.title)
	m.form[fProject].SetValue(v.project)
	m.form[fTag].SetValue(v.tag)
	m.form[fDate].SetValue(v.date)
	m.form[fDir].SetValue(v.dir)
	m.origNotes = v.notes
	m.form[fNotes].SetValue(flatten(v.notes))
	m.focusField(fTitle)
	m.form[fTitle].CursorEnd()
}

// expandDirField rewrites a leading ~ or $VAR in the dir field to the real path
// as you leave it, so you can see what will be saved.
func (m *Model) expandDirField() {
	if m.formIdx != fDir {
		return
	}
	if d, err := parse.ExpandPath(m.form[fDir].Value()); err != nil {
		m.status = "error: " + err.Error()
	} else {
		m.status = ""
		m.form[fDir].SetValue(d)
	}
}

func (m *Model) focusField(i int) {
	m.formIdx = (i + len(m.form)) % len(m.form)
	for j := range m.form {
		if j == m.formIdx {
			m.form[j].Focus()
		} else {
			m.form[j].Blur()
		}
	}
}

// submitForm saves the add/edit form; on a bad date it reports the error and keeps the form open.
func (m *Model) submitForm() {
	// Everything typed here is literal: +project, @tag and due: are CLI quick-add
	// syntax and mean nothing in the TUI.
	t := model.Task{
		Title:   strings.TrimSpace(m.form[fTitle].Value()),
		Project: strings.TrimSpace(m.form[fProject].Value()),
		Dir:     strings.TrimSpace(m.form[fDir].Value()), // expanded below
		Notes:   strings.TrimSpace(m.form[fNotes].Value()),
	}
	for _, g := range strings.FieldsFunc(m.form[fTag].Value(), func(r rune) bool { return r == ',' || r == ' ' }) {
		t.Tags = append(t.Tags, g)
	}
	if t.Notes == flatten(m.origNotes) {
		t.Notes = m.origNotes // untouched: keep original line breaks
	}
	// Leading ~ and $VAR in the directory are expanded; everything else is literal.
	var err error
	if t.Dir, err = parse.ExpandPath(t.Dir); err == nil {
		m.form[fDir].SetValue(t.Dir)
	}
	if d := strings.TrimSpace(m.form[fDate].Value()); d != "" && err == nil {
		var due time.Time
		if due, err = parse.Date(d, time.Now()); err == nil {
			t.Due = &due
		}
	}
	if err == nil && t.Title == "" {
		err = fmt.Errorf("a task needs a title")
	}
	if err == nil {
		if m.kind == inEdit {
			// The form holds every editable field, so it replaces them outright;
			// completion state is kept.
			old, ok := m.current()
			if !ok {
				m.mode = modeNormal
				return
			}
			t.ID = old.ID
			err = m.st.Update(&t)
		} else {
			err = m.st.Add(&t)
		}
	}
	if err != nil {
		m.status = "error: " + err.Error()
		return
	}
	m.mode = modeNormal
	m.reload()
}

// searchText is what the "/" filter matches against.
func searchText(t model.Task) string {
	return strings.Join(append([]string{t.Title, t.Project, t.Dir, t.Notes}, t.Tags...), " ")
}

func (m *Model) updateInput(msg tea.KeyMsg) (tea.Model, tea.Cmd) {
	if m.kind != inFilter {
		switch msg.String() {
		case "esc", "ctrl+c":
			m.mode, m.status = modeNormal, ""
		case "tab":
			m.expandDirField()
			m.focusField(m.formIdx + 1)
		case "shift+tab":
			m.expandDirField()
			m.focusField(m.formIdx - 1)
		case "enter":
			m.submitForm()
		default:
			var cmd tea.Cmd
			m.form[m.formIdx], cmd = m.form[m.formIdx].Update(msg)
			return m, cmd
		}
		return m, textinput.Blink
	}
	switch msg.String() {
	case "esc", "ctrl+c":
		m.mode = modeNormal
		return m, nil
	case "enter":
		m.mode = modeNormal
		m.submit(m.input.Value())
		return m, nil
	}
	var cmd tea.Cmd
	m.input, cmd = m.input.Update(msg)
	return m, cmd
}

func (m *Model) submit(value string) {
	m.text = strings.TrimSpace(value)
	m.cursor = 0
	m.reload()
}

func (m *Model) updateConfirm(msg tea.KeyMsg) (tea.Model, tea.Cmd) {
	if k := msg.String(); k == "y" || k == "Y" {
		if t, ok := m.current(); ok {
			m.check(m.st.Delete(t.ID))
		}
	}
	m.mode = modeNormal
	return m, nil
}

var (
	selected   = lipgloss.NewStyle().Reverse(true)
	headerText = render.Bold.Foreground(lipgloss.Color("6"))
)

func (m *Model) View() string {
	if m.w == 0 {
		return "loading..."
	}
	bodyH := max(3, m.h-2)
	if m.mode == modeInput && m.kind != inFilter {
		bodyH = max(3, m.h-5) // the form takes four footer lines
	}
	listW := max(20, m.w-sidebarWidth-3)

	var side []string
	for i, v := range m.views {
		line := " " + v.label
		switch {
		case v.header:
			if len(side) > 0 {
				side = append(side, "") // blank line before each category
			}
			line = headerText.Render(" " + strings.ToUpper(v.label))
		case i == m.viewIdx && !m.focusList:
			line = selected.Render(pad(line, sidebarWidth-1))
		case i == m.viewIdx:
			line = render.Bold.Render(line)
		}
		side = append(side, line)
	}
	sidebar := lipgloss.NewStyle().Width(sidebarWidth).Height(bodyH).MaxHeight(bodyH).
		Render(strings.Join(side, "\n"))

	var body string
	if m.showHelp {
		body = helpText
	} else {
		body = m.renderList(listW, bodyH)
	}
	main := lipgloss.NewStyle().Width(listW).Height(bodyH).MaxHeight(bodyH).Render(body)
	divider := render.Dim.Render(strings.TrimSuffix(strings.Repeat("│\n", bodyH), "\n"))
	top := lipgloss.JoinHorizontal(lipgloss.Top, sidebar, divider, " ", main)

	var foot string
	switch {
	case m.mode == modeInput && m.kind != inFilter:
		row := strings.Join([]string{m.form[fProject].View(), m.form[fTag].View(), m.form[fDate].View()}, "  ")
		foot = m.form[fTitle].View() + "\n" + row + "\n" + m.form[fDir].View() + "\n" + m.form[fNotes].View()
	case m.mode == modeInput:
		foot = m.input.View()
	case m.mode == modeConfirm:
		t, _ := m.current()
		foot = render.Red.Render(fmt.Sprintf("Delete %q? (y/N)", t.Title))
	case m.status != "":
		foot = render.Red.Render(m.status)
	default:
		hint := "? help"
		if m.text != "" {
			hint = fmt.Sprintf("filter: %q · ? help", m.text)
		}
		foot = render.Dim.Render(hint)
	}
	mid := ""
	if m.mode == modeInput {
		mid = render.Red.Render(m.status) // form errors show above the form
	}
	return top + "\n" + mid + "\n" + foot
}

func pad(s string, w int) string {
	if n := w - lipgloss.Width(s); n > 0 {
		return s + strings.Repeat(" ", n)
	}
	return s
}

func (m *Model) renderList(w, h int) string {
	if len(m.tasks) == 0 {
		return render.Dim.Render("No tasks. Press a to add one.")
	}
	now := time.Now()
	var lines []string
	cursorLine, cursorEnd := 0, 0
	last := -1
	grouped := m.views[m.viewIdx].kind != vDone // the Done tab is a flat list, newest first
	for i, t := range m.tasks {
		if b := bucket(t, now); grouped && b != last {
			if last != -1 {
				lines = append(lines, "")
			}
			lines = append(lines, headerText.Render(groupNames[b]))
			last = b
		}
		if i == m.cursor {
			cursorLine = len(lines)
		}
		lines = append(lines, m.taskLine(t, i == m.cursor && m.focusList, w, now))
		if m.expanded[t.ID] {
			lines = append(lines, detailLines(t, w, now)...)
			lines = append(lines, "") // blank line below the expansion
		}
		if i == m.cursor {
			cursorEnd = len(lines) - 1
		}
	}
	// Scroll so the cursor line stays visible.
	start := 0
	if cursorEnd >= h { // show the whole expanded block if it fits, else at least the task line
		start = min(cursorLine, cursorEnd-h+1)
	}
	end := min(len(lines), start+h)
	return strings.Join(lines[start:end], "\n")
}

// detailLines is the expanded view of a task: project, due date, tags and
// the wrapped description.
func detailLines(t model.Task, w int, now time.Time) []string {
	const indent = "      "
	row := func(label, value string) string {
		if value == "" {
			value = render.Dim.Render("none")
		}
		return indent + render.Dim.Render(fmt.Sprintf("%-8s", label)) + value
	}
	project, due, tags := "", "", ""
	if t.Project != "" {
		project = t.Project
	}
	if t.Due != nil {
		due = render.DueStyle(t.Due, now).Render(t.Due.Format("Mon 2006-01-02"))
	}
	if len(t.Tags) > 0 {
		tags = strings.Join(t.Tags, ", ")
	}
	out := []string{row("project", project), row("due", due), row("tags", tags), row("dir", t.Dir)}

	text := strings.TrimSpace(t.Notes)
	if text == "" {
		return append(out, row("notes", ""))
	}
	wrapped := lipgloss.NewStyle().Width(max(10, w-len(indent)-8)).Render(text)
	for k, l := range strings.Split(wrapped, "\n") {
		label := ""
		if k == 0 {
			label = "notes"
		}
		out = append(out, indent+render.Dim.Render(fmt.Sprintf("%-8s", label))+l)
	}
	return out
}

func (m *Model) taskLine(t model.Task, sel bool, w int, now time.Time) string {
	box := "[ ]"
	if t.Done() {
		box = "[x]"
	}
	meta := []string{}
	if t.Due != nil {
		meta = append(meta, render.RelativeLabel(t.Due, now))
	}
	// project and tags live in the expanded view (enter)
	right := strings.Join(meta, " ")
	left := fmt.Sprintf("%s %s", box, t.Title)
	room := w - lipgloss.Width(right) - 1
	if lipgloss.Width(left) > room && room > 4 {
		left = lipgloss.NewStyle().MaxWidth(room-1).Render(left) + "…"
	}
	line := pad(left, max(room, 0)) + " " + right
	line = lipgloss.NewStyle().MaxWidth(w).Render(line)
	if sel {
		return selected.Render(pad(line, w))
	}
	switch {
	case t.Done():
		return render.Strike.Render(line)
	case t.Due != nil && model.Day(*t.Due).Before(model.Day(now)):
		return render.Red.Render(line)
	}
	return line
}

const helpText = `Keys:

  hjkl         movement
  tab          jump between groups and categories
  space, x     mark done
  a            add task
  enter        expand / collapse the task's details
  e            edit task 
  d            delete task
  c            open a shell in the task's directory (exit to come back)
  /            search
  ?            help
  q            quit
	`
