package sim

import (
	"context"
	"fmt"
	"io"
	"time"
)

// WatchOptions controls the periodic trend/analysis writer.
type WatchOptions struct {
	Interval time.Duration
	Limit    int
	Once     bool
	Writer   io.Writer
}

// RunWatch periodically refreshes trend and analysis outputs until ctx is done.
// It does not launch new simulations; it is a scheduler for generated reports.
func RunWatch(ctx context.Context, paths Paths, opts WatchOptions) error {
	if opts.Interval <= 0 {
		opts.Interval = 10 * time.Minute
	}
	if opts.Limit <= 0 {
		opts.Limit = 50
	}
	run := func() error {
		trend, err := WriteTrend(paths, opts.Limit)
		if err != nil {
			return err
		}
		brief, err := WriteAnalysisBrief(paths, opts.Limit)
		if err != nil {
			return err
		}
		if opts.Writer != nil {
			fmt.Fprintf(opts.Writer, "trend: %s\nbrief: %s\n", trend.HTML, brief.Markdown)
		}
		return nil
	}
	if opts.Once {
		return run()
	}
	if err := run(); err != nil {
		return err
	}
	ticker := time.NewTicker(opts.Interval)
	defer ticker.Stop()
	for {
		select {
		case <-ctx.Done():
			return ctx.Err()
		case <-ticker.C:
			if err := run(); err != nil {
				return err
			}
		}
	}
}
