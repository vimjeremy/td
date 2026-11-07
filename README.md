# td

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
