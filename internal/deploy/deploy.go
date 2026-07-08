package deploy

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
)

func Run(ctx context.Context, opts Options, out io.Writer) error {
	result, err := Plan(opts)
	if err != nil {
		return err
	}
	if !opts.DryRun && result.Exposure == "internet" {
		if err := requireExistingTailscaleDeployment(ctx, result); err != nil {
			return err
		}
	}
	if err := writeResult(out, result, opts.JSON, opts.DryRun); err != nil {
		return err
	}
	if opts.DryRun {
		return nil
	}
	if opts.JSON {
		fmt.Fprintln(out)
	}
	return Apply(ctx, result, opts)
}

func writeResult(out io.Writer, result Result, asJSON bool, dryRun bool) error {
	if asJSON {
		encoder := json.NewEncoder(out)
		encoder.SetIndent("", "  ")
		return encoder.Encode(result)
	}
	if _, err := fmt.Fprint(out, result.Manifest); err != nil {
		return err
	}
	if !dryRun {
		return nil
	}
	if _, err := fmt.Fprintln(out, "\nCommands:"); err != nil {
		return err
	}
	for _, command := range result.Commands {
		if _, err := fmt.Fprintf(out, "  %s\n", command); err != nil {
			return err
		}
	}
	return nil
}
