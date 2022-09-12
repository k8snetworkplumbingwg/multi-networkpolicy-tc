package main

import (
	"context"
	"log"
	"os"
	"time"

	"github.com/spf13/cobra"
	"github.com/spf13/pflag"

	"k8s.io/apimachinery/pkg/util/wait"
	"k8s.io/klog/v2"

	"github.com/k8snetworkplumbingwg/multi-networkpolicy-tc/pkg/server"
	"github.com/k8snetworkplumbingwg/multi-networkpolicy-tc/pkg/utils"
)

const logFlushFreqFlagName = "log-flush-frequency"

var logFlushFreq = pflag.Duration(logFlushFreqFlagName, 5*time.Second, "Maximum number of seconds between log flushes")

// KlogWriter serves as a bridge between the standard log package and the glog package.
type KlogWriter struct{}

// Write implements the io.Writer interface.
func (writer KlogWriter) Write(data []byte) (n int, err error) {
	klog.InfoDepth(1, string(data))
	return len(data), nil
}

func initLogs(ctx context.Context) {
	log.SetOutput(KlogWriter{})
	log.SetFlags(0)
	go wait.Until(klog.Flush, *logFlushFreq, ctx.Done())
}

func main() {
	ctx := utils.SetupSignalHandler()
	initLogs(ctx)
	opts := server.NewOptions()

	cmd := &cobra.Command{
		Use:  "multi-networkpolicy-tc",
		Long: `TBD`,
		Run: func(cmd *cobra.Command, args []string) {
			srv, err := server.NewServer(opts)
			if err != nil {
				klog.Exit(err)
			}

			srv.Run(ctx)
		},
	}
	opts.AddFlags(cmd.Flags())

	if err := cmd.Execute(); err != nil {
		klog.Flush()
		os.Exit(1)
	}
	klog.Flush()
}
