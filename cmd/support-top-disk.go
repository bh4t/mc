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
	"log"
	"os"
	"time"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/minio/cli"
	"github.com/minio/madmin-go"
	"github.com/minio/mc/pkg/probe"
)

var supportTopDiskFlags = []cli.Flag{
	cli.IntFlag{
		Name:  "count, c",
		Usage: "show up to N disks",
		Value: 10,
	},
}

var supportTopDiskCmd = cli.Command{
	Name:            "disk",
	Usage:           "show current disk statistics",
	Action:          mainSupportTopDisk,
	OnUsageError:    onUsageError,
	Before:          setGlobalsFromContext,
	Flags:           append(supportTopDiskFlags, globalFlags...),
	HideHelpCommand: true,
	CustomHelpTemplate: `NAME:
  {{.HelpName}} - {{.Usage}}

USAGE:
  {{.HelpName}} [FLAGS] TARGET

FLAGS:
  {{range .VisibleFlags}}{{.}}
  {{end}}
EXAMPLES:
   1. Display disks metrics
      {{.Prompt}} {{.HelpName}} myminio/

`,
}

// checkSupportTopDiskSyntax - validate all the passed arguments
func checkSupportTopDiskSyntax(ctx *cli.Context) {
	if len(ctx.Args()) == 0 || len(ctx.Args()) > 1 {
		cli.ShowCommandHelpAndExit(ctx, "disk", 1) // last argument is exit code
	}
}

func mainSupportTopDisk(ctx *cli.Context) error {
	checkSupportTopDiskSyntax(ctx)

	aliasedURL := ctx.Args().Get(0)

	// Create a new MinIO Admin Client
	client, err := newAdminClient(aliasedURL)
	if err != nil {
		fatalIf(err.Trace(aliasedURL), "Unable to initialize admin client.")
		return nil
	}

	ctxt, cancel := context.WithCancel(globalContext)
	defer cancel()

	info, e := client.ServerInfo(ctxt)
	fatalIf(probe.NewError(e).Trace(aliasedURL), "Unable to initialize admin client.")

	var disks []madmin.Disk
	for _, srv := range info.Servers {
		disks = append(disks, srv.Disks...)
	}

	// MetricsOptions are options provided to Metrics call.
	opts := madmin.MetricsOptions{
		Type:     madmin.MetricsDisk,
		Interval: time.Second,
		ByDisk:   true,
	}

	done := make(chan struct{})

	p := tea.NewProgram(initTopDiskUI(disks, ctx.Int("count")))
	go func() {
		if e := p.Start(); e != nil {
			os.Exit(1)
		}
		close(done)
	}()

	go func() {
		out := func(m madmin.RealtimeMetrics) {
			for name, metric := range m.ByDisk {
				p.Send(topDiskResult{
					diskName: name,
					stats:    metric.IOStats,
				})
			}
		}

		e := client.Metrics(ctxt, opts, out)
		if e != nil {
			log.Fatalln(probe.NewError(e), "Unable to fetch top disks")
		}
	}()

	<-done
	return nil
}
