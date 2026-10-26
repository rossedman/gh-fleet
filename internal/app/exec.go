package app

import (
	"context"
	"errors"
	"fmt"
	"os"
	"sort"
	"strings"
	"time"

	"github.com/rossedman/gh-fleet/internal/campaign"
	"github.com/rossedman/gh-fleet/internal/engine"
	"github.com/rossedman/gh-fleet/internal/render"
	"github.com/rossedman/gh-fleet/internal/state"
)

// ExecOptions configures Exec.
type ExecOptions struct {
	Common
	// Argv is the command to run in each repo.
	Argv []string
	// Campaign is a campaign file or name to take targets from.
	Campaign string
	// Ad-hoc targets, used instead of a campaign file when any is set.
	Orgs      []string
	Topics    []string
	Languages []string
	Files     []string
	Search    string
	// Raw prints every repo's output instead of a tally.
	Raw bool
	// Timeout per repo.
	Timeout time.Duration
}

// adHoc reports whether ad-hoc target flags were given.
func (o ExecOptions) adHoc() bool {
	return len(o.Orgs) > 0 || len(o.Topics) > 0 || len(o.Languages) > 0 || len(o.Files) > 0 || o.Search != ""
}

// Tally is one distinct output line and the repos that printed it.
type Tally struct {
	Line  string   `json:"line"`
	Count int      `json:"count"`
	Repos []string `json:"repos"`
}

// ExecReport is the JSON form of gh fleet exec.
type ExecReport struct {
	Repos []state.RepoState `json:"repos"`
	Tally []Tally           `json:"tally"`
}

// Exec runs a read-only command in a fresh clone of every target and
// aggregates the output. It never pushes or opens PRs.
func (a *App) Exec(ctx context.Context, o ExecOptions) error {
	if len(o.Argv) == 0 {
		return errors.New("no command given; put it after --, e.g. gh fleet exec -- cat go.mod")
	}
	camp := &campaign.Campaign{Name: "exec"}
	repos := o.Repos
	_, haveFile := os.Stat(campaign.DefaultFile)
	if o.adHoc() || (o.Campaign == "" && haveFile != nil) {
		camp.Targets = campaign.Targets{Orgs: o.Orgs, Include: campaign.Include{
			Topics: o.Topics, Languages: o.Languages, Files: o.Files, Search: o.Search, Repos: o.Repos,
		}, Exclude: campaign.Exclude{Repos: o.Exclude}}
		repos = nil
		if len(camp.Targets.Orgs) == 0 && len(camp.Targets.Include.Repos) == 0 {
			return errors.New("no targets: pass --org or --repo, or run from a directory with a fleet.yaml")
		}
	} else {
		loaded, err := a.LoadCampaign(Common{Campaign: o.Campaign, Exclude: o.Exclude})
		if err != nil {
			return err
		}
		camp.Targets = loaded.Targets
		camp.Change.Env = loaded.Change.Env
		camp.Vars = loaded.Vars
	}
	camp.ApplyDefaults()
	client, err := a.NewClient(o.WriteRate)
	if err != nil {
		return err
	}
	res, err := a.resolve(ctx, o.Common, client, camp, repos, false)
	if err != nil {
		return err
	}
	p := a.printer(o.Common)
	progress := a.errPrinter(o.Common)
	pipe := &engine.Pipeline{Mode: engine.Exec, Campaign: camp, Workspaces: a.workspaces(o.Common), Git: a.Git, Argv: o.Argv, ExecTimeout: o.Timeout}
	// Exec lists only failures as they happen; the tally comes at the end.
	tr, stop := a.track(o.Common, "Running in", len(res.Targets), concurrency(o.Common, camp), false)
	results, fatal := engine.Run(ctx, res.Targets, pipe, engine.Options{
		Concurrency: concurrency(o.Common, camp),
		OnResult: func(rs state.RepoState) {
			if !p.Machine() {
				tr.Pause(func() {
					if o.Raw {
						for _, l := range strings.Split(strings.TrimRight(rs.Output, "\n"), "\n") {
							if l != "" {
								fmt.Fprintf(p.Out, "%s %s\n", p.Styles.Brand.Render(rs.Repo+":"), l)
							}
						}
					}
					if rs.Status == state.Failed {
						progress.Progress(rs)
					}
				})
			}
			tr.Result(rs)
		},
	})
	stop()
	sortStates(results)
	tally := Aggregate(results)
	if p.Machine() {
		if err := p.PrintJSON(ExecReport{Repos: results, Tally: tally}); err != nil {
			return err
		}
	} else {
		if !o.Raw {
			rows := make([][]string, len(tally))
			for i, t := range tally {
				rows[i] = []string{fmt.Sprint(t.Count), t.Line}
			}
			if len(rows) > 0 {
				p.Table([]string{"REPOS", "OUTPUT"}, rows, func(_, col int, s string) string {
					if col == 0 {
						return p.Styles.Brand.Render(s)
					}
					return s
				})
				fmt.Fprintln(p.Out)
			}
		}
		fmt.Fprintln(progress.Out, progress.Summary(render.Count(results)))
	}
	if fatal != nil {
		return fatal
	}
	return failures(results, "")
}

// Aggregate tallies distinct output lines across repos, counting each repo
// once per line, most common first.
func Aggregate(results []state.RepoState) []Tally {
	by := map[string]map[string]bool{}
	for _, r := range results {
		for _, l := range strings.Split(r.Output, "\n") {
			l = strings.TrimSpace(l)
			if l == "" {
				continue
			}
			if by[l] == nil {
				by[l] = map[string]bool{}
			}
			by[l][r.Repo] = true
		}
	}
	out := make([]Tally, 0, len(by))
	for line, repos := range by {
		t := Tally{Line: line, Count: len(repos)}
		for r := range repos {
			t.Repos = append(t.Repos, r)
		}
		sort.Strings(t.Repos)
		out = append(out, t)
	}
	sort.Slice(out, func(i, j int) bool {
		if out[i].Count != out[j].Count {
			return out[i].Count > out[j].Count
		}
		return out[i].Line < out[j].Line
	})
	return out
}
