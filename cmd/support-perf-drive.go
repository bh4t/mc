// Copyright (c) 2015-2022 MinIO, Inc.
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

	tea "github.com/charmbracelet/bubbletea"
	humanize "github.com/dustin/go-humanize"
	"github.com/minio/cli"
	"github.com/minio/madmin-go"
	"github.com/minio/mc/pkg/probe"
)

func mainAdminSpeedTestDrive(ctx *cli.Context, aliasedURL string) error {
	client, perr := newAdminClient(aliasedURL)
	if perr != nil {
		fatalIf(perr.Trace(aliasedURL), "Unable to initialize admin client.")
		return nil
	}

	ctxt, cancel := context.WithCancel(globalContext)
	defer cancel()

	blocksize, e := humanize.ParseBytes(ctx.String("blocksize"))
	if e != nil {
		fatalIf(probe.NewError(e), "Unable to parse blocksize")
		return nil
	}
	if blocksize < 0 {
		fatalIf(errInvalidArgument(), "blocksize cannot be <= 0")
		return nil
	}

	filesize, e := humanize.ParseBytes(ctx.String("filesize"))
	if e != nil {
		fatalIf(probe.NewError(e), "Unable to parse filesize")
		return nil
	}
	if filesize < 0 {
		fatalIf(errInvalidArgument(), "filesize cannot be <= 0")
		return nil
	}

	serial := ctx.Bool("serial")

	resultCh, speedTestErr := client.DriveSpeedtest(ctxt, madmin.DriveSpeedTestOpts{
		Serial:    serial,
		BlockSize: uint64(blocksize),
		FileSize:  uint64(filesize),
	})

	if globalJSON {
		if speedTestErr != nil {
			printMsg(speedTestResult{
				Type:  driveSpeedTest,
				Err:   speedTestErr.Error(),
				Final: true,
			})
			return nil
		}

		var results []madmin.DriveSpeedTestResult
		for result := range resultCh {
			if result.Version != "" {
				results = append(results, result)
			}
		}
		printMsg(speedTestResult{
			Type:        driveSpeedTest,
			DriveResult: results,
			Final:       true,
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
			printMsg(speedTestResult{
				Type: driveSpeedTest,
				Err:  speedTestErr.Error(),
			})
			return
		}

		var results []madmin.DriveSpeedTestResult
		for result := range resultCh {
			if result.Version != "" {
				results = append(results, result)
			} else {
				p.Send(speedTestResult{
					Type:        driveSpeedTest,
					DriveResult: []madmin.DriveSpeedTestResult{},
				})
			}
		}
		p.Send(speedTestResult{
			Type:        driveSpeedTest,
			DriveResult: results,
			Final:       true,
		})
	}()

	<-done

	return nil
}
