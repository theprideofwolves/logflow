package main

import (
	"context"
	"encoding/json"
	"flag"
	"fmt"
	"io"
	"os"
	"os/signal"
	"time"

	"github.com/theprideofwolves/logflow/internal/analysis"
	"github.com/theprideofwolves/logflow/internal/assistant"
)

func run(args []string, out, errs io.Writer) int {
	fs := flag.NewFlagSet("logflow", flag.ContinueOnError)
	fs.SetOutput(errs)
	window := fs.Duration("window", 10*time.Minute, "Inclusive event-time window")
	threshold := fs.Int("threshold", 6, "Failures needed to alert")
	workers := fs.Int("workers", 4, "Parser workers 1-32")
	replay := fs.Bool("replay", false, "Run offline synthetic detection and evasion scenarios")
	assistFile := fs.String("assist", "", "Read an exported investigation JSON and draft an evidence-grounded investigation")
	alertID := fs.String("alert", "", "Alert ID within the exported report")
	provider := fs.String("provider", "offline", "Assistant: offline, ollama or openai")
	allowRemote := fs.Bool("allow-remote", false, "Explicitly allow sending assistant context to OpenAI")
	mcpMode := fs.Bool("mcp", false, "Serve read-only snapshot tools over stdio; requires -assist and -alert")
	if e := fs.Parse(args); e != nil {
		if e == flag.ErrHelp {
			return 0
		}
		return 2
	}
	if *assistFile != "" {
		invalid := false
		fs.Visit(func(f *flag.Flag) {
			if f.Name == "threshold" || f.Name == "window" || f.Name == "workers" {
				invalid = true
			}
		})
		if invalid {
			fmt.Fprintln(errs, "analysis options cannot change an exported investigation")
			return 2
		}
		if fs.NArg() != 0 || *replay {
			fmt.Fprintln(errs, "assistant mode does not accept log input or replay")
			return 2
		}
		f, err := os.Open(*assistFile)
		if err != nil {
			fmt.Fprintln(errs, "cannot open report export")
			return 2
		}
		defer f.Close()
		b, err := io.ReadAll(io.LimitReader(f, 16<<20+1))
		if err != nil || len(b) > 16<<20 {
			fmt.Fprintln(errs, "report exceeds 16 MiB")
			return 2
		}
		var state saved
		if json.Unmarshal(b, &state) != nil {
			fmt.Fprintln(errs, "invalid report export")
			return 2
		}
		if state.Report == nil {
			var report analysis.Report
			if json.Unmarshal(b, &report) != nil {
				fmt.Fprintln(errs, "invalid CLI report")
				return 2
			}
			state.Report = &report
		}
		session, err := assistant.Open(state.Report, *alertID)
		if err != nil {
			fmt.Fprintln(errs, err)
			return 2
		}
		if *mcpMode {
			if *provider != "offline" || *allowRemote {
				fmt.Fprintln(errs, "MCP exposes reads only; no provider configuration")
				return 2
			}
			if err = assistant.ServeMCP(context.Background(), session); err != nil {
				fmt.Fprintln(errs, err)
				return 2
			}
			return 0
		}
		if *provider == "openai" && !*allowRemote {
			fmt.Fprintln(errs, "OpenAI requires -allow-remote")
			return 2
		}
		if *provider != "offline" && *provider != "ollama" && *provider != "openai" {
			fmt.Fprintln(errs, "unknown provider")
			return 2
		}
		draft := session.Build(context.Background(), *provider)
		if err = json.NewEncoder(out).Encode(draft); err != nil {
			return 2
		}
		if draft.Warning != "" {
			return 2
		}
		return 0
	}
	if *mcpMode || *alertID != "" || *provider != "offline" || *allowRemote {
		fmt.Fprintln(errs, "assistant options require -assist")
		return 2
	}
	if *replay {
		if fs.NArg() != 0 || fs.NFlag() != 1 {
			fmt.Fprintln(errs, "-replay must be used alone")
			return 2
		}
		results, err := analysis.Replay(context.Background())
		if err != nil {
			fmt.Fprintln(errs, err)
			return 2
		}
		if err = json.NewEncoder(out).Encode(results); err != nil {
			fmt.Fprintln(errs, err)
			return 2
		}
		for _, r := range results {
			if !r.MatchesExpectation {
				return 2
			}
		}
		return 0
	}
	if fs.NArg() != 1 {
		fmt.Fprintln(errs, "one JSONL file or - for stdin required")
		return 2
	}
	var input io.Reader = os.Stdin
	if fs.NArg() == 1 && fs.Arg(0) != "-" {
		f, e := os.Open(fs.Arg(0))
		if e != nil {
			fmt.Fprintln(errs, e)
			return 2
		}
		defer f.Close()
		input = f
	}
	opts := analysis.DefaultOptions()
	opts.Window = *window
	opts.Threshold = *threshold
	opts.Workers = *workers
	ctx, stop := signal.NotifyContext(context.Background(), os.Interrupt)
	defer stop()
	r, e := analysis.Process(ctx, input, opts)
	if e != nil {
		fmt.Fprintln(errs, e)
		return 2
	}
	if e = json.NewEncoder(out).Encode(r); e != nil {
		fmt.Fprintln(errs, e)
		return 2
	}
	if !r.Complete {
		return 2
	}
	return 0
}
func main() {
	if len(os.Args) == 1 || (len(os.Args) == 2 && os.Args[1] == "-ui") {
		if e := launchUI(); e != nil {
			fmt.Println(e)
			fmt.Println("Press Enter to close")
			fmt.Scanln()
			os.Exit(2)
		}
		return
	}
	os.Exit(run(os.Args[1:], os.Stdout, os.Stderr))
}
