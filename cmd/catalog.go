package cmd

import (
	"fmt"
	"github.com/posener/complete"
	log "github.com/sirupsen/logrus"
	"github.com/snowdrop/spring-boot-cloud-devex/pkg/catalog"
	"github.com/spf13/cobra"
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

			catalog.List(getK8RestConfig(), matching)
		},
	}
	catalogListCmd.Flags().StringVarP(&matching, "search", "s", "", "Only return services whose name matches the specified text")

	catalogInstanceCmd := &cobra.Command{
		Use:     "create",
		Short:   "Create a service instance",
		Long:    "Create a service instance and install it in a namespace.",
		Example: ` sd catalog create <instance name>`,
		Args:    cobra.ExactArgs(1),
		Run: func(cmd *cobra.Command, args []string) {
			log.Info("Catalog select command called")
			setup := Setup()

			catalog.Create(setup.RestConfig, setup.Application, args[0])
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

			catalog.Plan(getK8RestConfig(), args[0])
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

type classSuggester struct{}

func (i classSuggester) Predict(args complete.Args) []string {
	serviceCatalogClient := catalog.GetClient(getK8RestConfig())
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
