package cmd

import (
	"fmt"
	"github.com/ghodss/yaml"
	scv1beta1 "github.com/kubernetes-incubator/service-catalog/pkg/apis/servicecatalog/v1beta1"
	"github.com/manifoldco/promptui"
	"github.com/posener/complete"
	log "github.com/sirupsen/logrus"
	"github.com/snowdrop/spring-boot-cloud-devex/pkg/catalog"
	"github.com/spf13/cobra"
	"io/ioutil"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"sort"
)

// A component represents
type Component struct {
	metav1.TypeMeta   `json:",inline"`
	metav1.ObjectMeta `json:"metadata"`
	Spec              ComponentSpec `json:"spec"`
}

type ComponentSpec struct {
	// To indicate if we want to expose the service out side of the cluster as a route
	ExposeService bool      `json:"exposeService,omitempty"`
	Services      []Service `json:"services,omitempty"`
	// DeploymentMode indicates the strategy to be adopted to install the resources into a namespace
	// and next to create a pod. 2 strategies are currently supported; inner and outer loop
	// where outer loop refers to a build of the code and the packaging of the application into a container's image
	// while the inner loop will install a pod's running a supervisord daemon used to trigger actions such as : assemble, run, ...
	DeploymentMode string `json:"deployment,omitempty"`
	// Runtime is the framework used to start within the container the application
	// It corresponds to one of the following values: spring-boot, vertx, tornthail, nodejs
	Runtime string `json:"runtime,omitempty"`
	Version string `json:"version,omitempty"`
	// Array of env variables containing extra/additional info to be used to configure the runtime
	Envs []Env `json:"envs,omitempty"`
}

type Service struct {
	Class      string      `json:"class,omitempty"`
	Name       string      `json:"name,omitempty"`
	Plan       string      `json:"plan,omitempty"`
	ExternalId string      `json:"externalid,omitempty"`
	Parameters []Parameter `json:"parameters,omitempty"`
}

type Parameter struct {
	Name  string `json:"name,omitempty"`
	Value string `json:"value,omitempty"`
}

type Env struct {
	Name  string `json:"name,omitempty"`
	Value string `json:"value,omitempty"`
}

func init() {
	var matching string
	catalogListCmd := &cobra.Command{
		Use:     "list",
		Short:   "List all available services from the catalog",
		Long:    "List all available services from the Service Catalog's broker.",
		Example: ` sd catalog list [-s <part of service name>]`,
		Run: func(cmd *cobra.Command, args []string) {
			log.Info("Catalog list command called")

			catalog.List(GetK8RestConfig(), matching)
		},
	}
	catalogListCmd.Flags().StringVarP(&matching, "search", "s", "", "Only return services whose name matches the specified text")

	catalogInstanceCmd := &cobra.Command{
		Use:     "create",
		Short:   "Create a service instance",
		Long:    "Create a service instance and install it in a namespace.",
		Example: ` sd catalog create <instance name>`,
		//Args:    cobra.ExactArgs(1),
		Run: func(cmd *cobra.Command, args []string) {
			log.Info("Catalog create command called")

			client := catalog.GetClient(GetK8RestConfig())
			classesByCategory, _ := catalog.GetServiceClassesByCategory(client)

			prompt := promptui.Select{
				Label: "Which kind of service do you wish to create?",
				Items: catalog.GetServiceClassesCategories(classesByCategory),
			}

			_, category, _ := prompt.Run()

			templates := &promptui.SelectTemplates{
				Active:   "\U00002620 {{ .Name | cyan }}",
				Inactive: "  {{ .Name | cyan }}",
				Selected: "\U00002620 {{ .Name | red | cyan }}",
				Details: `
--------- Service Class ----------
{{ "Name:" | faint }}	{{ .Name }}
{{ "Description:" | faint }}	{{ .Description }}
{{ "Long:" | faint }}	{{ .LongDescription }}`,
			}

			uiClasses := getUiServiceClasses(classesByCategory[category])
			prompt = promptui.Select{
				Label:     "Which " + category + " service class should we use?",
				Items:     uiClasses,
				Templates: templates,
			}

			i, _, _ := prompt.Run()
			uiClass := uiClasses[i]
			className := uiClass.Name
			class := uiClass.Class

			plans, _ := catalog.GetMatchingPlans(client, class)
			prompt = promptui.Select{
				Label: "Which service plan should we use?",
				Items: catalog.GetServicePlanNames(plans),
			}

			_, planName, _ := prompt.Run()

			plan := plans[planName]

			properties, _ := catalog.GetProperties(plan)

			i = 0
			values := make(map[string]string)
			parameters := make([]Parameter, 0, len(properties))
			for i < len(properties) && properties[i].Required {
				prop := properties[i]
				prompt := promptui.Prompt{
					Label:     fmt.Sprintf("Enter a value for %s property %s", prop.Type, prop.Title),
					AllowEdit: true,
				}

				result, _ := prompt.Run()
				values[prop.Name] = result
				parameters = append(parameters, Parameter{
					Name:  prop.Name,
					Value: result,
				})
				i++
			}

			// if we have non-required properties, ask if user wants to provide values
			if i < len(properties)-1 {
				// todo
			}

			instancePrompt := promptui.Prompt{
				Label:     "Finally, how should we name your service",
				AllowEdit: true,
			}

			instance, _ := instancePrompt.Run()

			//setup := Setup()

			/*err := catalog.CreateServiceInstance(client, setup.Application.Namespace, instance, className, planName, "", values)
			if err != nil {
				panic(err)
			}*/

			/*
				apiVersion: component.k8s.io/v1alpha1
				kind: Component
				metadata:
				  name: my-spring-boot-service
				spec:
				  services:
				    - name: my-postgresql-db   # Name of the instance to be created within the namespace
				      class: dh-postgresql-apb # Class or name of the service selected from the catalog
				      plan: dev                # Plan selected : dev, ....
				      parameters:
				      - name: postgresql_user
				        value: "luke"
				      - name: postgresql_password
				        value: "secret"
				      - name: postgresql_database
				        value: "my_data"
				      - name: postgresql_version
				        value: "9.6"
			*/
			component := Component{
				TypeMeta: metav1.TypeMeta{
					Kind:       "Component",
					APIVersion: "component.k8s.io/v1alpha1",
				},
				ObjectMeta: metav1.ObjectMeta{
					Name: "my-spring-boot-service",
				},
				Spec: ComponentSpec{
					Services: []Service{
						{Name: instance,
							Class:      className,
							Plan:       planName,
							Parameters: parameters,
						},
					},
				},
			}
			b, err := yaml.Marshal(component)
			if err != nil {
				log.Fatal(err)
			}
			err = ioutil.WriteFile("component-service.yml", b, 0644)
			if err != nil {
				log.Fatal(err)
			}

			log.Infof("Service %s using class %s has been created!", instance, className)
		},
	}

	var (
		secret   string
		instance string
	)
	catalogBindCmd := &cobra.Command{
		Use:     "bind",
		Short:   "Bind a service to a secret's namespace",
		Long:    "Bind a service to a secret's namespace.",
		Example: ` sd catalog bind --secret foo --toInstance instance`,
		Run: func(cmd *cobra.Command, args []string) {
			log.Info("Catalog Bind command called")
			setup := Setup()

			catalog.Bind(setup.RestConfig, setup.Application, instance, secret)
			catalog.MountSecretAsEnvFrom(setup.RestConfig, setup.Application, secret)
		},
	}
	catalogBindCmd.Flags().StringVarP(&secret, "secret", "s", "", "Name of the secret to bind")
	catalogBindCmd.Flags().StringVarP(&instance, "toInstance", "i", "", "Instance name to bind the secret to")

	catalogPlanCmd := &cobra.Command{
		Use:     "plan",
		Short:   "Show the plans of a service",
		Long:    "Show the plans of a ClusterServiceClass",
		Example: ` sd catalog plan <class name>`,
		Args:    cobra.ExactArgs(1),
		Run: func(cmd *cobra.Command, args []string) {
			log.Info("Catalog plan command called")

			catalog.Plan(GetK8RestConfig(), args[0])
		},
	}
	Suggesters[GetCommandSuggesterName(catalogPlanCmd)] = classSuggester{}

	catalogCmd := &cobra.Command{
		Use:   "catalog [options]",
		Short: "List, select or bind a service from a catalog.",
		Long:  `List, select or bind a service from a catalog.`,
		Example: fmt.Sprintf("%s\n%s\n%s\n%s",
			catalogListCmd.Example,
			catalogInstanceCmd.Example,
			catalogPlanCmd.Example,
			catalogBindCmd.Example),
	}

	catalogCmd.AddCommand(catalogListCmd)
	catalogCmd.AddCommand(catalogInstanceCmd)
	catalogCmd.AddCommand(catalogPlanCmd)
	catalogCmd.AddCommand(catalogBindCmd)

	catalogCmd.Annotations = map[string]string{"command": "catalog"}
	rootCmd.AddCommand(catalogCmd)
}

type uiServiceClasses []catalog.UIClusterServiceClass

func (classes uiServiceClasses) Len() int {
	return len(classes)
}

func (classes uiServiceClasses) Less(i, j int) bool {
	return classes[i].Name < classes[j].Name
}

func (classes uiServiceClasses) Swap(i, j int) {
	classes[i], classes[j] = classes[j], classes[i]
}
func getUiServiceClasses(classes []scv1beta1.ClusterServiceClass) (uiClasses uiServiceClasses) {
	uiClasses = make(uiServiceClasses, 0, len(classes))
	for _, v := range classes {
		uiClasses = append(uiClasses, catalog.ConvertToUI(v))
	}

	sort.Sort(uiClasses)
	return uiClasses
}

type classSuggester struct{}

func (i classSuggester) Predict(args complete.Args) []string {
	serviceCatalogClient := catalog.GetClient(GetK8RestConfig())
	classes, err := catalog.GetClusterServiceClasses(serviceCatalogClient)

	if err != nil {
		log.Error(err)
	}

	var suggestions []string
	for _, class := range classes {
		suggestions = append(suggestions, class.Spec.ExternalName)
	}

	return suggestions
}
