package cmd

import (
	"github.com/spf13/cobra"
	log "github.com/sirupsen/logrus"

	"github.com/cmoulliard/k8s-supervisor/pkg/buildpack"

	corev1 "k8s.io/api/core/v1"

	"github.com/cmoulliard/k8s-supervisor/pkg/common/oc"
)

var (
	ports string
)
var debugCmd = &cobra.Command{
	Use:   "debug [flags]",
	Short: "Debug your SpringBoot's application",
	Long:  `Debug your SpringBoot's application.`,
	Example: ` sb debug -p 5005:5005`,
	Args: cobra.RangeArgs(0, 1),
	Run: func(cmd *cobra.Command, args []string) {

		log.Info("Debug command called")

		_, pod := SetupAndWaitForPod()
		podName := pod.Name

		// Append Debug Env Vars and update POD
		//log.Info("[Step 5] - Add new ENV vars for remote Debugging")
		//pod.Spec.Containers[0].Env = append(pod.Spec.Containers[0].Env,debugEnvVars()...)
		//clientset.CoreV1().Pods(application.Namespace).Update(pod)

		// Start Java Application
		supervisordBin := "/var/lib/supervisord/bin/supervisord"
		supervisordCtl := "ctl"
		cmdName := "run-java"

		log.Info("[Step 5] - Restart the Spring Boot application ...")
		oc.ExecCommand(oc.Command{Args: []string{"rsh",podName,supervisordBin,supervisordCtl,"stop",cmdName}})
		oc.ExecCommand(oc.Command{Args: []string{"rsh",podName,supervisordBin,supervisordCtl,"start",cmdName}})

		// Forward local to Remote port
		log.Info("[Step 6] - Remote Debug the spring Boot Application ...")
		oc.ExecCommand(oc.Command{Args: []string{"port-forward",podName,ports}})
	},
}

func init() {
	debugCmd.Flags().StringVarP(&ports,"ports","p","5005:5005","Local and remote ports to be used to forward trafic between the dpo and your machine.")
	//debugCmd.MarkFlagRequired("ports")

	debugCmd.Annotations = map[string]string{"command": "debug"}
	rootCmd.AddCommand(debugCmd)
}

func debugEnvVars() []corev1.EnvVar {
	return []corev1.EnvVar{
		{
			Name: "JAVA_DEBUG",
			Value: "true",
		},
		{
			Name: "JAVA_DEBUG_PORT",
			Value: "5005",
		},
	}
}

