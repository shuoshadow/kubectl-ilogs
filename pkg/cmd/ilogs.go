package cmd

import (
	"fmt"
	log "github.com/sirupsen/logrus"
	"github.com/spf13/cobra"
	"k8s.io/cli-runtime/pkg/genericclioptions"
	"k8s.io/client-go/rest"
	"k8s.io/client-go/tools/clientcmd"
	"kubectl-ilogs/pkg/ilogs"
)

var (
	ilogsLong = `
ILogs is an interactive pod and container selector for 'kubectl logs'

Arg[1] will act as a filter, any pods that match will be returned in a list
that the user can select from.
`
	ilogsExample = `
	# select from all pods in the namespace then run: 'kubectl logs '
	%[1]s ilogs 

	# select from all pods matching [busybox] then run: 'kubectl logs -f [pod_name]'
	%[1]s ilogs busybox

	# select from all pods matching [multi_container_pod]
	# then select from all containers in pod matching [second_container]
	# then run: 'kubectl logs -f [pod_name] -c [container_name]'
	%[1]s ilogs multi_container_pod -c second_container
`
)

type ILogsOptions struct {
	configFlags *genericclioptions.ConfigFlags
	clientCfg *rest.Config

	configOverrides clientcmd.ConfigOverrides
	allNamespaces bool
	containerFilter string
	lvl string
	namespace string
	naked bool
	vimMode bool

	genericclioptions.IOStreams
}

func NewILogsOptions(streams genericclioptions.IOStreams) *ILogsOptions {
	return &ILogsOptions{
		configFlags: genericclioptions.NewConfigFlags(true),

		IOStreams: streams,
	}
}

func NewCmdILogs(streams genericclioptions.IOStreams) *cobra.Command {
	o := NewILogsOptions(streams)

	cmd := &cobra.Command{
		Use: "ilogs [pod filter] [flags]",
		Short: "Logs Kubernetes Pod",
		Args: cobra.MinimumNArgs(1),
		Example: fmt.Sprintf(ilogsExample, "kubectl"),
		Long: ilogsLong,
		SilenceUsage: false,
		RunE: func(cmd *cobra.Command, args []string) error {
			if err := o.Complete(cmd, args); err != nil {
				return err
			}

			if err := o.Run(args); err != nil {
				return err
			}

			return nil
		},
	}

	cmd.Flags().BoolVarP(&o.allNamespaces, "all-namespaces", "A", o.allNamespaces, "If present, list the requested object(s) across all namespaces. Namespace in current context is ignored even if specified with --namespace.")
	cmd.PersistentFlags().StringVarP(&o.containerFilter, "container", "c", "", "Container to search")
	cmd.PersistentFlags().StringVarP(&o.lvl, "log-level", "l", "", "log level (trace|debug|info|warn|error|fatal|panic)")
	cmd.PersistentFlags().BoolVarP(&o.vimMode, "vim-mode", "v", false, "Vim Mode enabled")
	cmd.PersistentFlags().BoolVarP(&o.naked, "naked", "x", false, "Decolorize output")
	// 将k8s client的config 绑定到 cmd的flagset
	o.configFlags.AddFlags(cmd.Flags())

	return cmd
}

func (o *ILogsOptions) Complete(cmd *cobra.Command, args []string) error {
	var err error

	o.clientCfg, err = o.configFlags.ToRESTConfig()
	if err != nil {
		return err
	}

	c := clientcmd.NewNonInteractiveDeferredLoadingClientConfig(
		clientcmd.NewDefaultClientConfigLoadingRules(),
		&o.configOverrides,
		)

	o.namespace, _, err = c.Namespace()
	if err != nil {
		return err
	}

	if *o.configFlags.Namespace != "" {
		o.namespace = *o.configFlags.Namespace
	}

	if o.allNamespaces {
		o.namespace = ""
	}

	switch o.lvl {
	case "trace":
		log.SetLevel(log.TraceLevel)
	case "debug":
		log.SetLevel(log.DebugLevel)
	case "info":
		log.SetLevel(log.InfoLevel)
	case "warn":
		log.SetLevel(log.WarnLevel)
	case "error":
		log.SetLevel(log.ErrorLevel)
	case "fatal":
		log.SetLevel(log.FatalLevel)
	case "panic":
		log.SetLevel(log.PanicLevel)
	default:
		log.SetLevel(log.ErrorLevel)
	}

	return nil
}

func (o *ILogsOptions) Run(args []string) error {
	podFilter := args[0]

	config := &ilogs.Config{
		Namespace: o.namespace,
		Naked: o.naked,
		VimMode: o.vimMode,
		PodFilter: podFilter,
		ContainerFilter: o.containerFilter,
	}

	r := ilogs.NewIlogs(o.clientCfg, config)

	if err := r.Do(); err != nil {
		return err
	}

	return nil
}