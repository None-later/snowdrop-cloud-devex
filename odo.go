package main

import (
	"flag"
	"fmt"

	"github.com/golang/glog"

	"k8s.io/client-go/kubernetes"
	"k8s.io/client-go/tools/clientcmd"

	// "github.com/cmoulliard/k8s-odo-supervisor/pkg/signals"

	restclient "k8s.io/client-go/rest"
	appsocpv1 "github.com/openshift/client-go/apps/clientset/versioned/typed/apps/v1"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

var (
	masterURL  string
	kubeconfig string

	filter = metav1.ListOptions{
		LabelSelector: "io.openshift.odo=inject-supervisord",
	}
)

const (
	namespace = "k8s-supervisord"
)

/*func main() {
	flag.Parse()

	// set up signals so we handle the first shutdown signal gracefully
	stopCh := signals.SetupSignalHandler()

	// Build kube config using kube config folder on the developer's machine
	cfg, err := clientcmd.BuildConfigFromFlags(masterURL, kubeconfig)
	if err != nil {
		glog.Fatalf("Error building kubeconfig: %s", err.Error())
	}

	kubeClient, err := kubernetes.NewForConfig(cfg)
	if err != nil {
		glog.Fatalf("Error building kubernetes clientset: %s", err.Error())
	}

	fmt.Println("Kube config parsed correctly")

	controller := NewController(kubeClient,cfg, namespace)

	if err = controller.Run(2, stopCh); err != nil {
		glog.Fatalf("Error running controller: %s", err.Error())
	}

}*/

func main() {
	flag.Parse()

	// Build kube config using kube config folder on the developer's machine
	cfg, err := clientcmd.BuildConfigFromFlags(masterURL, kubeconfig)
	if err != nil {
		glog.Fatalf("Error building kubeconfig: %s", err.Error())
	}

	_, err = kubernetes.NewForConfig(cfg)
	if err != nil {
		glog.Fatalf("Error building kubernetes clientset: %s", err.Error())
	}

	fmt.Println("Fetching about DC to be injected")
	findDeploymentconfig(cfg)
}

func init() {
	flag.StringVar(&kubeconfig, "kubeconfig", "", "Path to a kubeconfig. Only required if out-of-cluster.")
	flag.StringVar(&masterURL, "master", "", "The address of the Kubernetes API server. Overrides any value in kubeconfig. Only required if out-of-cluster.")
}

func findDeploymentconfig(config *restclient.Config) {
	deploymentConfigV1client, err := appsocpv1.NewForConfig(config)
	if err != nil {
		glog.Error("")
	}

	deploymentList, err := deploymentConfigV1client.DeploymentConfigs(namespace).List(filter)
	fmt.Printf("Listing deployments in namespace %s: \n", namespace)
	if err != nil {
		glog.Error("Error to get Deployment Config !")
	}
	for _, d := range deploymentList.Items {
		fmt.Printf("%s\n", d.Name)
		d.Spec.Template = supervisordInitContainer()
		_, err := deploymentConfigV1client.DeploymentConfigs(namespace).Update(&d)
		if err != nil {
			glog.Error("Error to update the Deployment Config ! %s\n", err)
		}
		//fmt.Printf("Raw printout of the dc %+v\n", d)
	}
}

func supervisordInitContainer() *corev1.PodTemplateSpec {
	return &corev1.PodTemplateSpec{
		ObjectMeta: metav1.ObjectMeta{
			Labels: map[string]string{
				"app": "spring-boot-supervisord",
				"deploymentconfig": "spring-boot-supervisord",
			},
		},
		Spec: corev1.PodSpec{
			InitContainers: []corev1.Container{
				{
					Name:    "copy-supervisord",
					Image:   "docker/dd/dd",
					Command: []string{"/usr/bin/cp"},
					Args:    []string{"-r","/opt/supervisord"," /var/lib/"},
					VolumeMounts: []corev1.VolumeMount{
						{
						 Name: "shared-data",
						 MountPath: "/var/lib/supervisord",
						},
					},
				},
			},
		},
	}
}
