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
	if err := writeResult(out, result, opts.JSON); err != nil {
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

func writeResult(out io.Writer, result Result, asJSON bool) error {
	if asJSON {
		encoder := json.NewEncoder(out)
		encoder.SetIndent("", "  ")
		return encoder.Encode(result)
	}
	_, err := fmt.Fprint(out, result.Manifest)
	return err
}
