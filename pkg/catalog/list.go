package catalog

import (
	"encoding/json"
	scv1beta1 "github.com/kubernetes-incubator/service-catalog/pkg/apis/servicecatalog/v1beta1"
	restclient "k8s.io/client-go/rest"
	"os"
	"strings"

	log "github.com/sirupsen/logrus"

	"github.com/olekukonko/tablewriter"
	"sort"
)

func List(config *restclient.Config, matching string) {
	serviceCatalogClient := GetClient(config)
	classes, _ := GetClusterServiceClasses(serviceCatalogClient)
	log.Info("List of services")
	log.Info("================")

	var filtered []scv1beta1.ClusterServiceClass
	if len(matching) > 0 {
		filtered = make([]scv1beta1.ClusterServiceClass, 0)
		for _, value := range classes {
			if strings.Contains(value.Spec.CommonServiceClassSpec.ExternalName, matching) {
				filtered = append(filtered, value)
			}
		}
	} else {
		filtered = classes
	}

	sort.Slice(filtered[:], func(i, j int) bool {
		return filtered[i].Spec.ExternalName < filtered[j].Spec.ExternalName
	})

	printListResults(filtered)
}

func ConvertToUI(class scv1beta1.ClusterServiceClass) UIClusterServiceClass {
	var meta map[string]interface{}
	json.Unmarshal(class.Spec.ExternalMetadata.Raw, &meta)
	longDescription := ""
	if val, ok := meta["longDescription"]; ok {
		longDescription = val.(string)
	}
	return UIClusterServiceClass{
		Name:            class.Spec.ExternalName,
		Description:     class.Spec.Description,
		LongDescription: longDescription,
		Class:           class,
	}
}

type UIClusterServiceClass struct {
	Name            string
	Description     string
	LongDescription string
	Class           scv1beta1.ClusterServiceClass
}

func printListResults(classes []scv1beta1.ClusterServiceClass) {
	table := tablewriter.NewWriter(os.Stdout)
	table.SetRowLine(true)
	table.SetHeader([]string{"Name", "Description", "Long Description"})
	table.SetAlignment(tablewriter.ALIGN_LEFT)
	table.SetCenterSeparator("*")
	table.SetColumnSeparator("‡")
	table.SetRowSeparator("-")
	for _, class := range classes {
		uiClass := ConvertToUI(class)
		row := []string{uiClass.Name, uiClass.Description, uiClass.LongDescription}
		table.Append(row)
	}
	table.Render()
}
