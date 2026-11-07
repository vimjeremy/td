package cmd

import (
	"bufio"
	"fmt"
	"os"
	"strings"
	"time"

	"github.com/jeremy/td/internal/model"
	"github.com/jeremy/td/internal/parse"
	"github.com/jeremy/td/internal/render"
	"github.com/spf13/cobra"
)

func init() {
	rootCmd.AddCommand(addCmd, lsCmd, doneCmd, undoCmd, editCmd, rmCmd, showCmd)

	lsCmd.Flags().Bool("today", false, "only tasks due today")
	lsCmd.Flags().Bool("overdue", false, "only overdue tasks")
	lsCmd.Flags().Bool("week", false, "only tasks due in the next 7 days (including today)")
	lsCmd.Flags().Bool("all", false, "include completed tasks")
	lsCmd.Flags().Bool("done", false, "only completed tasks")
	editCmd.Flags().String("title", "", "replace the title")
	editCmd.Flags().String("notes", "", "replace the notes")
	rmCmd.Flags().BoolP("yes", "y", false, "do not ask for confirmation")
}

var addCmd = &cobra.Command{
	Use:   "add <text>",
	Short: "Add a task, e.g. td add Call dentist +personal @phone due:tomorrow",
	Args:  cobra.MinimumNArgs(1),
	RunE: func(cmd *cobra.Command, args []string) error {
		p, err := parse.Input(args, time.Now())
		if err != nil {
			return err
		}
		t, err := p.ToTask()
		if err != nil {
			return err
		}
		st, err := openStore()
		if err != nil {
			return err
		}
		defer st.Close()
		if err := st.Add(&t); err != nil {
			return err
		}
		fmt.Printf("Added #%d: %s\n", t.ID, t.Title)
		return nil
	},
}

var lsCmd = &cobra.Command{
	Use:   "ls [+project] [@tag]",
	Short: "List tasks",
	RunE: func(cmd *cobra.Command, args []string) error {
		now := time.Now()
		today := model.Day(now)
		var f model.Filter
		for _, a := range args {
			switch {
			case strings.HasPrefix(a, "+") && len(a) > 1:
				f.Project = a[1:]
			case strings.HasPrefix(a, "@") && len(a) > 1:
				f.Tag = a[1:]
			default:
				return fmt.Errorf("unexpected argument %q (use +project or @tag)", a)
			}
		}
		fl := cmd.Flags()
		if v, _ := fl.GetBool("today"); v {
			f.DueFrom, f.DueTo = &today, &today
		}
		if v, _ := fl.GetBool("overdue"); v {
			y := today.AddDate(0, 0, -1)
			f.DueTo = &y
		}
		if v, _ := fl.GetBool("week"); v {
			w := today.AddDate(0, 0, 7)
			f.DueFrom, f.DueTo = &today, &w
		}
		f.All, _ = fl.GetBool("all")
		f.DoneOnly, _ = fl.GetBool("done")

		st, err := openStore()
		if err != nil {
			return err
		}
		defer st.Close()
		tasks, err := st.List(f)
		if err != nil {
			return err
		}
		if len(tasks) == 0 {
			fmt.Println("No tasks.")
			return nil
		}
		render.Table(os.Stdout, tasks, now)
		return nil
	},
}

func setDone(done bool) func(*cobra.Command, []string) error {
	return func(cmd *cobra.Command, args []string) error {
		ids, err := parseIDs(args)
		if err != nil {
			return err
		}
		st, err := openStore()
		if err != nil {
			return err
		}
		defer st.Close()
		for _, id := range ids {
			if err := st.SetDone(id, done); err != nil {
				return fmt.Errorf("#%d: %w", id, err)
			}
			verb := "Completed"
			if !done {
				verb = "Reopened"
			}
			fmt.Printf("%s #%d\n", verb, id)
		}
		return nil
	}
}

var doneCmd = &cobra.Command{Use: "done <id...>", Short: "Mark tasks complete", Args: cobra.MinimumNArgs(1), RunE: setDone(true)}
var undoCmd = &cobra.Command{Use: "undo <id...>", Short: "Reopen completed tasks", Args: cobra.MinimumNArgs(1), RunE: setDone(false)}

var editCmd = &cobra.Command{
	Use:   "edit <id> [text]",
	Short: "Change a task, e.g. td edit 4 due:mon (given fields replace existing ones)",
	Args:  cobra.MinimumNArgs(1),
	RunE: func(cmd *cobra.Command, args []string) error {
		ids, err := parseIDs(args[:1])
		if err != nil {
			return err
		}
		p, err := parse.Input(args[1:], time.Now())
		if err != nil {
			return err
		}
		st, err := openStore()
		if err != nil {
			return err
		}
		defer st.Close()
		t, err := st.Get(ids[0])
		if err != nil {
			return fmt.Errorf("#%d: %w", ids[0], err)
		}
		p.Apply(&t)
		if cmd.Flags().Changed("title") {
			t.Title, _ = cmd.Flags().GetString("title")
		}
		if cmd.Flags().Changed("notes") {
			t.Notes, _ = cmd.Flags().GetString("notes")
		}
		if strings.TrimSpace(t.Title) == "" {
			return fmt.Errorf("a task needs a title")
		}
		if err := st.Update(&t); err != nil {
			return err
		}
		fmt.Printf("Updated #%d: %s\n", t.ID, t.Title)
		return nil
	},
}

var rmCmd = &cobra.Command{
	Use:   "rm <id...>",
	Short: "Delete tasks",
	Args:  cobra.MinimumNArgs(1),
	RunE: func(cmd *cobra.Command, args []string) error {
		ids, err := parseIDs(args)
		if err != nil {
			return err
		}
		st, err := openStore()
		if err != nil {
			return err
		}
		defer st.Close()
		yes, _ := cmd.Flags().GetBool("yes")
		for _, id := range ids {
			t, err := st.Get(id)
			if err != nil {
				return fmt.Errorf("#%d: %w", id, err)
			}
			if !yes {
				fmt.Printf("Delete #%d %q? [y/N] ", id, t.Title)
				line, _ := bufio.NewReader(os.Stdin).ReadString('\n')
				if a := strings.ToLower(strings.TrimSpace(line)); a != "y" && a != "yes" {
					fmt.Println("Skipped.")
					continue
				}
			}
			if err := st.Delete(id); err != nil {
				return err
			}
			fmt.Printf("Deleted #%d\n", id)
		}
		return nil
	},
}

var showCmd = &cobra.Command{
	Use:   "show <id>",
	Short: "Show one task in full",
	Args:  cobra.ExactArgs(1),
	RunE: func(cmd *cobra.Command, args []string) error {
		ids, err := parseIDs(args)
		if err != nil {
			return err
		}
		st, err := openStore()
		if err != nil {
			return err
		}
		defer st.Close()
		t, err := st.Get(ids[0])
		if err != nil {
			return fmt.Errorf("#%d: %w", ids[0], err)
		}
		now := time.Now()
		fmt.Printf("#%d  %s\n", t.ID, render.Bold.Render(t.Title))
		row := func(k, v string) {
			if v != "" {
				fmt.Printf("  %-9s %s\n", k, v)
			}
		}
		row("project", t.Project)
		if len(t.Tags) > 0 {
			row("tags", "@"+strings.Join(t.Tags, " @"))
		}
		if t.Due != nil {
			row("due", render.DueStyle(t.Due, now).Render(t.Due.Format("Mon 2006-01-02")+" ("+render.DueLabel(t.Due, now)+")"))
		}
		row("created", t.CreatedAt.Local().Format("2006-01-02 15:04"))
		if t.CompletedAt != nil {
			row("done", t.CompletedAt.Local().Format("2006-01-02 15:04"))
		}
		row("notes", t.Notes)
		return nil
	},
}
