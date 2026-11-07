# td

<video controls muted width="100%">
  <source src="media/demo.webm" type="video/webm">
  <source src="media/demo.mp4" type="video/mp4">
</video>

[Watch the demo](media/demo.webm) if the video does not play above.

A fast terminal todo list written in Go for vim users. Add tasks with one command, or enter the TUI for a keyboard-driven full-screen view.

## Install

Requires Go.

```sh
make build
make install
```

## Usage

| Command | What it does |
|---|---|
| `td add <task>` | Add a task |
| `td ls [+project] [@tag]` | List tasks |
| `td show <id>` | Show the details of a task |
| `td edit <id> [text]` | Change a task's details |
| `td done <id>` / `td undo <id>` | Finish or reopen tasks |
| `td rm [-y] <id>` | Delete task |
| `td projects` / `td tags` | List projects/tags |
| `td` / `td ui` | Open the TUI |


## Quick-add syntax

`td add` and `td edit` read these tokens; everything else becomes the title.

| Token | Meaning |
|---|---|
| `+name` | Project (`+none` clears it) |
| `@name` | Tag; repeat for several |
| `due:<date>` | `today`, `tomorrow`, a weekday, `+3d`, `+2w` or `2026-12-01` (`due:none` clears) |
| `dir:<path>` | Directory the task relates to |
| `note: <text>` | Notes; takes the rest of the line, so put it last |


Example:
```sh
td add Call dentist +personal @phone due:tomorrow dir:none note: ask about the invoice
```
