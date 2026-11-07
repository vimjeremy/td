# td

A fast terminal todo list written in Go for vim users. Add tasks in one line from the shell, or press `td` for a keyboard-driven full-screen view.

- Due dates, projects, tags and notes
- Tasks grouped by when they are due: Today, Tomorrow, This week, Next week, ...
- Vim-style navigation (`hjkl`) in the interactive UI
- A single static binary; tasks live in one local SQLite file

## Install

Requires Go.

```sh
make build     # builds ./td
make install   # builds and copies td to ~/.local/bin
```

## Quick start

```sh
td add Call dentist +personal @phone due:tomorrow
td ls                # what is still to do
td done 1            # finish task 1
td                   # open the interactive UI
```

## Command line

### Quick-add syntax

Used by `td add` and `td edit`. Everything that is not a token becomes the title.

| Token | Meaning |
|---|---|
| `+name` | Project (`+none` clears it on edit) |
| `@name` | Tag; repeat for several |
| `due:<date>` | Due date, or `due:none` to clear it |

Dates: `today`, `tomorrow`, a weekday (`fri` is the next Friday after today), `+3d`, `+2w`, or `2026-12-01`.

### Commands

| Command | What it does |
|---|---|
| `td add <text>` | Add a task |
| `td ls [+project] [@tag]` | List open tasks, soonest first |
| `td ls --today \| --overdue \| --week` | Only tasks due today, overdue, or in the next 7 days |
| `td ls --all \| --done` | Include finished tasks, or show only those |
| `td show <id>` | Show one task in full |
| `td edit <id> [text]` | Change a task; only the fields you give are replaced. `--title` and `--notes` flags also work |
| `td done <id...>` / `td undo <id...>` | Finish or reopen tasks |
| `td rm [-y] <id...>` | Delete tasks (asks first unless `-y`) |
| `td projects` / `td tags` | List them with open-task counts |
| `td` or `td ui` | Open the interactive UI |

## Interactive UI

A sidebar on the left (All, Today, Upcoming, Done, then your PROJECTS and TAGS) and the task list on the right.

- **All** shows open and finished tasks; **Upcoming** shows everything still to be done; **Done** shows finished tasks, newest first.
- Project and tag tabs stay until their last task is deleted.
- The list is grouped by due date: Overdue, Today, Tomorrow, This week, Next week, This month, Next month, This year, Next year, Later, No date, Done. Weeks run Monday to Sunday.
- Press `Enter` on a task to expand it and see its project, due date, tags and notes.

### Keys

| Key | Action |
|---|---|
| `j` / `k`, arrows | Move down / up (`g` / `G`: top / bottom) |
| `h` / `l` | Focus the sidebar / the list |
| `Tab`, `Shift+Tab` | Sidebar: jump between the main tabs, PROJECTS and TAGS. List: jump between date groups |
| `Space`, `x` | Mark done (reopens a task in the Done tab) |
| `Enter` | Expand or collapse the task |
| `a` | Add a task |
| `e` | Edit the task |
| `d` | Delete the task (asks to confirm) |
| `/` | Filter by text (`Esc` clears it) |
| `?` | Help |
| `q` | Quit |

### Add and edit form

`Tab` / `Shift+Tab` cycle title → project → tag → date → notes, `Enter` saves and `Esc` cancels. Text is taken literally: `+`, `@` and `due:` are command-line syntax only. Tags are separated by spaces or commas, and the date field takes the same values as `due:`. Adding from a project or tag tab prefills that project or tag.

## Data

Tasks are stored in a SQLite file:

1. `$TD_DB`, if set
2. else `$XDG_DATA_HOME/td/td.db`
3. else `~/.local/share/td/td.db`

Colors are turned off automatically when output is not a terminal or `NO_COLOR` is set.

## Development

```sh
make test      # go vet + go test
make build
```

| Path | Contents |
|---|---|
| `cmd/` | Cobra commands |
| `internal/store/` | SQLite access and schema migrations |
| `internal/parse/` | Quick-add and date parsing |
| `internal/render/` | Table output and colors for `td ls` |
| `internal/ui/` | Bubble Tea interface and date grouping |
| `internal/model/` | Shared types |
