package cmd

import (
	log "github.com/sirupsen/logrus"
	"github.com/snowdrop/k8s-supervisor/pkg/scaffold"
	"github.com/spf13/cobra"

	"archive/zip"
	"fmt"
	"io"
	"io/ioutil"
	"net/http"
	"net/url"
	"os"
	"path/filepath"
	"strings"
)

type SpringForm struct {
	GroupId string
}

func init() {
	var (
		templates = []string{"simple", "rest", "crud"}
		p         = scaffold.Project{}
	)

	createCmd := &cobra.Command{
		Use:     "create [flags]",
		Short:   "Create a Spring Boot maven project",
		Long:    `Create a Spring Boot maven project.`,
		Example: ` sb create`,
		Args:    cobra.RangeArgs(0, 1),
		Run: func(cmd *cobra.Command, args []string) {
			var valid bool
			for _, t := range templates {
				if p.Template == t {
					valid = true
				}
			}

			if !valid {
				log.WithField("template", p.Template).Fatal("The provided template is not supported: ")
			}

			log.Infof("Create command called")

			client := http.Client{}

			form := url.Values{}
			form.Add("template", p.Template)
			form.Add("groupid", p.GroupId)
			form.Add("artifactid", p.ArtifactId)
			form.Add("version", p.Version)
			form.Add("packagename", p.PackageName)
			form.Add("snowdropbom", p.SnowdropBomVersion)
			form.Add("springbootversion", p.SpringBootVersion)
			form.Add("outdir", p.OutDir)
			for _, v := range p.Modules {
				if v != "" {
					form.Add("module", v)
				}
			}

			parameters := form.Encode()
			if parameters != "" {
				parameters = "?" + parameters
			}

			u := strings.Join([]string{p.UrlService, "app"}, "/") + parameters
			log.Infof("URL of the request calling the service is %s", u)
			req, err := http.NewRequest(http.MethodGet, u, strings.NewReader(""))

			if err != nil {
				log.Error(err.Error())
			}
			addClientHeader(req)

			res, err := client.Do(req)
			if err != nil {
				log.Error(err.Error())
			}
			body, err := ioutil.ReadAll(res.Body)
			if err != nil {
				log.Error(err.Error())
			}

			currentDir, _ := os.Getwd()
			dir := filepath.Join(currentDir, p.OutDir)
			zipFile := dir + ".zip"

			err = ioutil.WriteFile(zipFile, body, 0644)
			if err != nil {
				log.Errorf("Failed to download file %s due to %s", zipFile, err)
			}
			err = Unzip(zipFile, dir)
			if err != nil {
				log.Errorf("Failed to unzip new project file %s due to %s", zipFile, err)
			}
			err = os.Remove(zipFile)
			if err != nil {
				log.Errorf(err.Error())
			}

		},
	}

	createCmd.Flags().StringVarP(&p.Template, "template", "t", "simple",
		fmt.Sprintf("Template name used to select the project to be created. Supported templates are '%s'", strings.Join(templates, ",")))
	createCmd.Flags().StringVarP(&p.UrlService, "urlservice", "u", "http://spring-boot-generator.195.201.87.126.nip.io", "URL of the HTTP Server exposing the spring boot service")
	createCmd.Flags().StringArrayVarP(&p.Modules, "module", "m", []string{}, "Spring Boot modules/starters")
	createCmd.Flags().StringVarP(&p.GroupId, "groupid", "g", "com.example", "Group ID")
	createCmd.Flags().StringVarP(&p.ArtifactId, "artifactid", "i", "demo", "Artifact ID")
	createCmd.Flags().StringVarP(&p.Version, "version", "v", "1.0", "Version")
	createCmd.Flags().StringVarP(&p.PackageName, "packagename", "p", "com.example.demo", "Package Name")
	createCmd.Flags().StringVarP(&p.SpringBootVersion, "springbootversion", "s", "1.5.15.RELEASE", "Spring Boot Version")
	createCmd.Flags().StringVarP(&p.SnowdropBomVersion, "snowdropbom", "b", "1.5.15.Final", "Snowdrop Bom Version")

	// Add a defined annotation in order to appear in the help menu
	createCmd.Annotations = map[string]string{"command": "create"}

	rootCmd.AddCommand(createCmd)
}

func addClientHeader(req *http.Request) {
	// TODO Define a version
	userAgent := "sb/1.0"
	req.Header.Set("User-Agent", userAgent)
}

func Unzip(src, dest string) error {
	r, err := zip.OpenReader(src)
	if err != nil {
		return err
	}
	defer r.Close()

	for _, f := range r.File {
		rc, err := f.Open()
		if err != nil {
			return err
		}
		defer rc.Close()

		name := filepath.Join(dest, f.Name)
		if f.FileInfo().IsDir() {
			os.MkdirAll(name, os.ModePerm)
		} else {
			var fdir string
			if lastIndex := strings.LastIndex(name, string(os.PathSeparator)); lastIndex > -1 {
				fdir = name[:lastIndex]
			}

			err = os.MkdirAll(fdir, os.ModePerm)
			if err != nil {
				return err
			}
			f, err := os.OpenFile(
				name, os.O_WRONLY|os.O_CREATE|os.O_TRUNC, f.Mode())
			if err != nil {
				return err
			}
			defer f.Close()

			_, err = io.Copy(f, rc)
			if err != nil {
				return err
			}
		}
	}
	return nil
}
