package buildpack

import (
	log "github.com/sirupsen/logrus"

	restclient "k8s.io/client-go/rest"

	appsocpv1 "github.com/openshift/client-go/apps/clientset/versioned/typed/apps/v1"
	appsv1 "github.com/openshift/api/apps/v1"

	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/api/resource"


	"github.com/cmoulliard/k8s-supervisor/pkg/buildpack/types"
	"k8s.io/client-go/kubernetes"
)

func CreatePVC(clientset *kubernetes.Clientset, application types.Application, size string) {
	quantity, err := resource.ParseQuantity(size)
	if err != nil {
		log.Fatal(err.Error())
	}

	pvc := &corev1.PersistentVolumeClaim{
		ObjectMeta: metav1.ObjectMeta{
			Name:   "m2-data",
			Labels: map[string]string{
				"app": application.Name,
			},
		},
		Spec: corev1.PersistentVolumeClaimSpec{
			Resources: corev1.ResourceRequirements{
				Requests: corev1.ResourceList{
					corev1.ResourceStorage: quantity,
				},
			},
			AccessModes: []corev1.PersistentVolumeAccessMode{
				corev1.ReadWriteOnce,
			},
		},
	}

	_, errPVC := clientset.CoreV1().PersistentVolumeClaims(application.Namespace).Create(pvc)
	if errPVC != nil {
		log.Fatal(errPVC.Error())
	}
}

func CreateDeploymentConfig(config *restclient.Config, application types.Application) *appsv1.DeploymentConfig {
	deploymentConfigV1client, err := appsocpv1.NewForConfig(config)
	if err != nil {
		log.Fatalf("Can't get DeploymentConfig Clientset: %s", err.Error())
	}

	dc, errCreate := deploymentConfigV1client.DeploymentConfigs(application.Namespace).Create(javaDeploymentConfig(application))
	if errCreate != nil {
		log.Fatalf("DeploymentConfig not created: %s", errCreate.Error())
	}
	return dc
}

func javaDeploymentConfig(application types.Application) *appsv1.DeploymentConfig {
	return &appsv1.DeploymentConfig{
		ObjectMeta: metav1.ObjectMeta{
			Name: application.Name,
			Labels: map[string]string{
				"app":              application.Name,
				"io.openshift.odo": "inject-supervisord",
			},
		},
		Spec: appsv1.DeploymentConfigSpec{
			Replicas: 1,
			Selector: map[string]string{
				"app":              application.Name,
				"deploymentconfig": application.Name,
			},
			Strategy: appsv1.DeploymentStrategy{
				Type: appsv1.DeploymentStrategyTypeRolling,
			},
			Template: &corev1.PodTemplateSpec{
				ObjectMeta: metav1.ObjectMeta{
					Name: application.Name,
					Labels: map[string]string{
						"app":              application.Name,
						"deploymentconfig": application.Name,
					},
				},
				Spec: corev1.PodSpec{
					InitContainers: []corev1.Container{*supervisordInitContainer()},
					Containers: []corev1.Container{
						{
							Image: appImagename + ":latest",
							Name:  application.Name,
							Ports: []corev1.ContainerPort{
								{
									ContainerPort: application.Port,
									Protocol:      corev1.ProtocolTCP,
								},
							},
							Env: []corev1.EnvVar{
								{
									Name:  "JAVA_APP_DIR",
									Value: "/deployments",
								},
								{
									Name:  "JAVA_APP_JAR",
									Value: appImagename + "-" + version + ".jar",
								},
								{
									Name: "JAVA_DEBUG",
									Value: "true",
								},
								{
									Name: "JAVA_DEBUG_PORT",
									Value: "5005",
								},
							},
							/*							Resources: corev1.ResourceRequirements{
															Limits: corev1.ResourceList{
																corev1.ResourceCPU: resource.MustParse(appConfig.Cpu),
																corev1.ResourceMemory: resource.MustParse(appConfig.Memory),
															},
														},*/
							VolumeMounts: []corev1.VolumeMount{
								{
									Name:      "shared-data",
									MountPath: "/var/lib/supervisord",
								},
								{
									Name:      "m2-data",
									MountPath: "/tmp/artifacts",
								},
							},
							Command: []string{
								"/var/lib/supervisord/bin/supervisord",
							},
							Args: []string{
								"-c",
								"/var/lib/supervisord/conf/supervisor.conf",
							},
						},
					},
					Volumes: []corev1.Volume{
						{
							Name: "shared-data",
							VolumeSource: corev1.VolumeSource{
								EmptyDir: &corev1.EmptyDirVolumeSource{},
							},
						},
						{
							Name: "m2-data",
							VolumeSource: corev1.VolumeSource{
								PersistentVolumeClaim: &corev1.PersistentVolumeClaimVolumeSource{
									ClaimName: "m2-data",
								},
							},
						},
					},
				},
			},
			Triggers: []appsv1.DeploymentTriggerPolicy{
				{
					Type: "ImageChange",
					ImageChangeParams: &appsv1.DeploymentTriggerImageChangeParams{
						Automatic: true,
						ContainerNames: []string{
							supervisordimagename,
						},
						From: corev1.ObjectReference{
							Kind: "ImageStreamTag",
							Name: supervisordimagename + ":latest",
						},
					},
				},
				{
					Type: "ImageChange",
					ImageChangeParams: &appsv1.DeploymentTriggerImageChangeParams{
						Automatic: true,
						ContainerNames: []string{
							application.Name,
						},
						From: corev1.ObjectReference{
							Kind: "ImageStreamTag",
							Name: appImagename + ":latest",
						},
					},
				},
			},
		},
	}
}

func supervisordInitContainer() *corev1.Container {
	return &corev1.Container{
		Name:    "copy-supervisord",
		Image:   supervisordimagename + ":latest",
		VolumeMounts: []corev1.VolumeMount{
			{
				Name:      "shared-data",
				MountPath: "/var/lib/supervisord",
			},
		},
		// TODO : The following list should be calculated based on the labels of the S2I image
		Env: []corev1.EnvVar{
			{
				Name:  "CMDS",
				Value: "echo:/var/lib/supervisord/conf/echo.sh;run-java:/usr/local/s2i/run;compile-java:/usr/local/s2i/assemble;build:/deployments/buildapp",
			},
		},
	}
}