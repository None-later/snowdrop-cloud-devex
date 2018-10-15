package cmd

import (
	"fmt"
	scv1beta1 "github.com/kubernetes-incubator/service-catalog/pkg/apis/servicecatalog/v1beta1"
	"github.com/manifoldco/promptui"
	"github.com/pkg/errors"
	"github.com/posener/complete"
	log "github.com/sirupsen/logrus"
	"github.com/snowdrop/spring-boot-cloud-devex/pkg/catalog"
	"github.com/spf13/cobra"
	"sort"
	"strings"
)

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

	var (
		serviceClass string
		planName     string
		instanceName string
		parameters   []string
	)
	catalogInstanceCmd := &cobra.Command{
		Use:     "create",
		Short:   "Create a service instance",
		Long:    "Create a service instance and install it in a namespace.",
		Example: ` sd catalog create`,
		Run: func(cmd *cobra.Command, args []string) {
			log.Info("Catalog create command called")

			client := catalog.GetClient(GetK8RestConfig())

			var uiClass catalog.UIClusterServiceClass
			if len(serviceClass) == 0 {
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
				uiClass = uiClasses[i]
			} else {
				class, _ := catalog.GetServiceClass(client, serviceClass)
				uiClass = catalog.ConvertToUI(class)
			}

			plans, _ := catalog.GetMatchingPlans(client, uiClass.Class)

			if len(planName) == 0 {
				prompt := promptui.Select{
					Label: "Which service plan should we use?",
					Items: catalog.GetServicePlanNames(plans),
				}
				_, planName, _ = prompt.Run()
			}

			plan, ok := plans[planName]
			if !ok {
				prompt := promptui.Select{
					Label: fmt.Sprintf("Unknown plan '%s'. Here are the valid options", planName),
					Items: catalog.GetServicePlanNames(plans),
				}
				_, planName, _ = prompt.Run()

				plan = plans[planName]
			}

			properties, _ := catalog.GetProperties(plan)

			var i = 0
			values := make(map[string]string)
			passedValues, err := getParametersAsMap(parameters)
			for i < len(properties) && properties[i].Required {
				prop := properties[i]
				if _, ok = passedValues[prop.Name]; !ok {
					prompt := promptui.Prompt{
						Label:     fmt.Sprintf("Enter a value for %s property %s", prop.Type, prop.Title),
						AllowEdit: true,
					}

					result, _ := prompt.Run()
					values[prop.Name] = result
				}

				i++
			}
			// if we have non-required properties, ask if user wants to provide values
			if i < len(properties)-1 {
				// todo
			}

			if len(instanceName) == 0 {
				instancePrompt := promptui.Prompt{
					Label:     "Finally, how should we name your service",
					AllowEdit: true,
				}
				instanceName, _ = instancePrompt.Run()
			}

			setup := Setup()

			err = catalog.CreateServiceInstance(client, setup.Application.Namespace, instanceName, uiClass.Name, planName, "", values)
			if err != nil {
				panic(err)
			}

			log.Infof("Service %s using class %s has been created!", instanceName, uiClass.Name)
		},
	}
	catalogInstanceCmd.Flags().StringVar(&serviceClass, "class", "", "Service class name")
	Suggesters[GetFlagSuggesterName(catalogInstanceCmd, "class")] = classSuggester{}
	catalogInstanceCmd.Flags().StringVar(&planName, "plan", "", "Plan name")
	catalogInstanceCmd.Flags().StringVar(&instanceName, "instance", "", "Instance name to use for the new service")
	catalogInstanceCmd.Flags().StringSliceVarP(&parameters, "parameters", "p", []string{}, "Comma-separated name=value pairs")

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

func getParametersAsMap(params []string) (parameters map[string]string, err error) {
	parameters = make(map[string]string, len(params))
	for _, value := range params {
		equals := strings.IndexRune(value, '=')
		if equals > 0 {
			split := strings.Split(value, "=")
			parameters[split[0]] = split[1]
		} else {
			return parameters, errors.Errorf("Invalid parameter, must follow 'name=value' format")
		}
	}
	return parameters, nil
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
