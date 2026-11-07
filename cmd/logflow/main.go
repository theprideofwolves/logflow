package main

import (
	"context"
	"encoding/json"
	"flag"
	"fmt"
	"github.com/theprideofwolves/logflow/internal/analysis"
	"io"
	"os"
	"os/signal"
	"time"
)

func run(args []string, out, errs io.Writer) int {
	fs := flag.NewFlagSet("logflow", flag.ContinueOnError)
	fs.SetOutput(errs)
	window := fs.Duration("window", 10*time.Minute, "Inclusive event-time window")
	threshold := fs.Int("threshold", 6, "Failures needed to alert")
	workers := fs.Int("workers", 4, "Parser workers 1-32")
	if e := fs.Parse(args); e != nil {
		if e == flag.ErrHelp {
			return 0
		}
		return 2
	}
	if fs.NArg() > 1 {
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
