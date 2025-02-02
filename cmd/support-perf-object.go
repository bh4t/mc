// Copyright (c) 2015-2021 MinIO, Inc.
//
// This file is part of MinIO Object Storage stack
//
// This program is free software: you can redistribute it and/or modify
// it under the terms of the GNU Affero General Public License as published by
// the Free Software Foundation, either version 3 of the License, or
// (at your option) any later version.
//
// This program is distributed in the hope that it will be useful
// but WITHOUT ANY WARRANTY; without even the implied warranty of
// MERCHANTABILITY or FITNESS FOR A PARTICULAR PURPOSE.  See the
// GNU Affero General Public License for more details.
//
// You should have received a copy of the GNU Affero General Public License
// along with this program.  If not, see <http://www.gnu.org/licenses/>.

package cmd

import (
	"context"
	"os"
	"time"

	tea "github.com/charmbracelet/bubbletea"
	humanize "github.com/dustin/go-humanize"
	"github.com/minio/cli"
	"github.com/minio/madmin-go"
	"github.com/minio/mc/pkg/probe"
	"github.com/minio/pkg/console"
)

var adminSpeedtestCmd = cli.Command{
	Name:               "speedtest",
	Usage:              "Run server side speed test",
	Action:             mainAdminSpeedtest,
	OnUsageError:       onUsageError,
	Before:             setGlobalsFromContext,
	HideHelpCommand:    true,
	Hidden:             true,
	CustomHelpTemplate: "Please use 'mc support perf'",
}

func mainAdminSpeedtest(ctx *cli.Context) error {
	console.Infoln("Please use 'mc support perf'")
	return nil
}

func mainAdminSpeedTestObject(ctx *cli.Context, aliasedURL string) error {
	client, perr := newAdminClient(aliasedURL)
	if perr != nil {
		fatalIf(perr.Trace(aliasedURL), "Unable to initialize admin client.")
		return nil
	}

	ctxt, cancel := context.WithCancel(globalContext)
	defer cancel()

	duration, e := time.ParseDuration(ctx.String("duration"))
	if e != nil {
		fatalIf(probe.NewError(e), "Unable to parse duration")
		return nil
	}
	if duration <= 0 {
		fatalIf(errInvalidArgument(), "duration cannot be 0 or negative")
		return nil
	}
	size, e := humanize.ParseBytes(ctx.String("size"))
	if e != nil {
		fatalIf(probe.NewError(e), "Unable to parse object size")
		return nil
	}
	if size < 0 {
		fatalIf(errInvalidArgument(), "size is expected to be atleast 0 bytes")
		return nil
	}
	concurrent := ctx.Int("concurrent")
	if concurrent <= 0 {
		fatalIf(errInvalidArgument(), "concurrency cannot be '0' or negative")
		return nil
	}
	globalPerfTestVerbose = ctx.Bool("verbose")

	// Turn-off autotuning only when "concurrent" is specified
	// in all other scenarios keep auto-tuning on.
	autotune := !ctx.IsSet("concurrent")

	resultCh, speedTestErr := client.Speedtest(ctxt, madmin.SpeedtestOpts{
		Size:        int(size),
		Duration:    duration,
		Concurrency: concurrent,
		Autotune:    autotune,
		Bucket:      ctx.String("bucket"), // This is a hidden flag.
	})

	if globalJSON {
		if speedTestErr != nil {
			printMsg(speedTestResult{
				Type:  objectSpeedTest,
				Err:   speedTestErr.Error(),
				Final: true,
			})
			return nil
		}

		var result madmin.SpeedTestResult
		for result = range resultCh {
			if result.Version == "" {
				continue
			}
			printMsg(speedTestResult{
				Type:         objectSpeedTest,
				ObjectResult: &result,
			})
		}

		printMsg(speedTestResult{
			Type:         objectSpeedTest,
			ObjectResult: &result,
			Final:        true,
		})

		return nil
	}

	done := make(chan struct{})

	p := tea.NewProgram(initSpeedTestUI())
	go func() {
		if e := p.Start(); e != nil {
			os.Exit(1)
		}
		close(done)
	}()

	go func() {
		if speedTestErr != nil {
			p.Send(speedTestResult{
				Type:  objectSpeedTest,
				Err:   speedTestErr.Error(),
				Final: true,
			})
			return
		}

		var result madmin.SpeedTestResult
		for result = range resultCh {
			p.Send(speedTestResult{
				Type:         objectSpeedTest,
				ObjectResult: &result,
			})
		}
		p.Send(speedTestResult{
			Type:         objectSpeedTest,
			ObjectResult: &result,
			Final:        true,
		})
	}()

	<-done

	return nil
}
