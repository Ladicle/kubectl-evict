package main

import (
	"fmt"
	"io"
	"os"

	"github.com/awslabs/operatorpkg/context"
	"github.com/spf13/cobra"
	"github.com/spf13/pflag"
	policyv1 "k8s.io/api/policy/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/cli-runtime/pkg/genericclioptions"
	cmdutil "k8s.io/kubectl/pkg/cmd/util"
)

var (
	// set values via build flags
	version string
	commit  string
)

type options struct {
	podNames []string

	dryRun             []string
	propagationPolicy  *metav1.DeletionPropagation
	gracePeriodSeconds *int64

	f cmdutil.Factory
}

func main() {
	var opts options
	var (
		dryRun       bool
		policy       string
		gracePeriods int64
	)
	cmd := &cobra.Command{
		Use:          "evict <POD_NAME>...",
		SilenceUsage: true,
		Version:      fmt.Sprintf("%s (%s)", version, commit),
		RunE: func(cmd *cobra.Command, args []string) error {
			if len(args) == 0 {
				return cmdutil.UsageErrorf(cmd, "evict requires POD_NAME")
			}
			opts.podNames = args

			if dryRun {
				opts.dryRun = append(opts.dryRun, metav1.DryRunAll)
			}
			if policy != "" {
				switch policy {
				case "Orphan", "Background", "Foreground":
					opts.propagationPolicy = (*metav1.DeletionPropagation)(&policy)
				default:
					return cmdutil.UsageErrorf(cmd, "invalid propagation policy: %s", policy)
				}
			}
			if gracePeriods >= 0 {
				opts.gracePeriodSeconds = &gracePeriods
			}

			return run(cmd.Context(), cmd.OutOrStdout(), opts)
		},
	}

	fsets := cmd.PersistentFlags()
	cfgFlags := genericclioptions.NewConfigFlags(true)
	cfgFlags.AddFlags(fsets)
	matchVersionFlags := cmdutil.NewMatchVersionFlags(cfgFlags)
	matchVersionFlags.AddFlags(fsets)
	fsets.VisitAll(func(f *pflag.Flag) { f.Hidden = true })

	opts.f = cmdutil.NewFactory(matchVersionFlags)

	cmd.Flags().BoolVar(&dryRun, "dry-run", false, "Enable the dray-run option.")
	cmd.Flags().StringVar(&policy, "propagation-policy", "", "Propagation policy for deleting the pod. Valid values are 'orphan', 'background' and 'foreground'.")
	cmd.Flags().Int64Var(&gracePeriods, "grace-period", -1, "Period of time in seconds given to the pod to terminate gracefully. Ignored if negative.")

	if err := cmd.Execute(); err != nil {
		os.Exit(1)
	}
}

func run(ctx context.Context, out io.Writer, opts options) error {
	k8sCfg := opts.f.ToRawKubeConfigLoader()
	namespace, _, err := k8sCfg.Namespace()
	if err != nil {
		return err
	}
	client, err := opts.f.KubernetesClientSet()
	if err != nil {
		return err
	}

	for _, name := range opts.podNames {
		if err := client.PolicyV1().Evictions(namespace).Evict(ctx, &policyv1.Eviction{
			ObjectMeta: metav1.ObjectMeta{
				Name:      name,
				Namespace: namespace,
			},
			DeleteOptions: &metav1.DeleteOptions{
				GracePeriodSeconds: opts.gracePeriodSeconds,
				PropagationPolicy:  opts.propagationPolicy,
				DryRun:             opts.dryRun,
			},
		}); err != nil {
			return err
		}
		fmt.Fprintf(out, "pod/%s evicted\n", name)
	}
	return nil
}
