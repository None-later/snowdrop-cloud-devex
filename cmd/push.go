package cmd

import (
	"os"
	"strings"
	"github.com/spf13/cobra"
	log "github.com/sirupsen/logrus"

	"github.com/cmoulliard/k8s-supervisor/pkg/buildpack"
	"github.com/cmoulliard/k8s-supervisor/pkg/common/config"
	"github.com/cmoulliard/k8s-supervisor/pkg/common/oc"

	"k8s.io/client-go/tools/clientcmd"
	"k8s.io/client-go/kubernetes"
)

var (
	mode string
	artefacts = []string{ "src", "pom.xml"}
)

var pushCmd = &cobra.Command{
	Use:   "push",
	Short: "Push local code to the development's pod",
	Long:  `Push local code to the development's pod.`,
	Example: ` sb push`,
	Args: cobra.RangeArgs(0, 1),
	Run: func(cmd *cobra.Command, args []string) {

		modeType := cmd.Flag("mode").Value.String();

		log.Info("sb Push command called")

		// Parse MANIFEST
		log.Info("[Step 1] - Parse MANIFEST of the project if it exists")
		current, _ := os.Getwd()
		application := buildpack.ParseManifest(current + "/MANIFEST")
		// Add Namespace's value
		application.Namespace = namespace

		// Get K8s' config file
		log.Info("[Step 2] - Get K8s config file")
		var kubeCfg = config.NewKube()
		if cmd.Flag("kubeconfig").Value.String() == "" {
			kubeCfg.Config = config.HomeKubePath()
		} else {
			kubeCfg.Config = cmd.Flag("kubeconfig").Value.String()
		}
		log.Debug("Kubeconfig : ",kubeCfg)

		// Create Kube Rest's Config Client
		log.Info("[Step 3] - Create kube Rest config client using config's file of the developer's machine")
		kubeRestClient, err := clientcmd.BuildConfigFromFlags(kubeCfg.MasterURL, kubeCfg.Config)
		if err != nil {
			log.Fatalf("Error building kubeconfig: %s", err.Error())
		}

		clientset, errclientset := kubernetes.NewForConfig(kubeRestClient)
		if errclientset != nil {
			log.Fatalf("Error building kubernetes clientset: %s", errclientset.Error())
		}

		// Wait till the dev's pod is available
		log.Info("[Step 4] - Wait till the dev's pod is available")
		pod, err := buildpack.WaitAndGetPod(clientset,application)
		if err != nil {
			log.Error("Pod watch error",err)
		}

		podName := pod.Name

		log.Info("[Step 5] - Copy files from the local developer's project to the pod")

		switch modeType {
		case "source":
			for i := range artefacts {
				log.Debug("Artefact : ",artefacts[i])
				oc.ExecCommand(oc.Command{Args: []string{"cp",oc.Client.Pwd + "/" + artefacts[i],podName+":/tmp/src/","-c","spring-boot-supervisord"}})
			}
		case "binary":
			uberjarName := strings.Join([]string{application.Name,application.Version},"-") +  ".jar"
			log.WithField("uberjarname",uberjarName).Debug("Uber jar name : ")
			oc.ExecCommand(oc.Command{Args: []string{"cp",oc.Client.Pwd + "/target/" + uberjarName,podName+":/deployments","-c","spring-boot-supervisord"}})
		default:
			log.WithField("mode",modeType).Fatal("The provided mode is not supported : ")
		}
	},
}

func init() {
	pushCmd.Flags().StringVarP(&mode,"mode","","","Source code or Binary compiled as uberjar within target directory")
	pushCmd.MarkFlagRequired("mode")
	pushCmd.Annotations = map[string]string{"command": "push"}

	rootCmd.AddCommand(pushCmd)
}

