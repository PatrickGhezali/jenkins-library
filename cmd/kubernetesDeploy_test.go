package cmd

import (
	"fmt"
	"github.com/stretchr/testify/assert"
	"testing"
)

func TestRunKubernetesDeploy(t *testing.T) {

	t.Run("test helm", func(t *testing.T) {
		opts := kubernetesDeployOptions{
			ContainerRegistryURL:      "https://my.registry:55555",
			ContainerRegistryUser:     "registryUser",
			ContainerRegistryPassword: "********",
			ChartPath:                 "path/to/chart",
			DeploymentName:            "deploymentName",
			DeployTool:                "helm",
			HelmDeployWaitSeconds:     400,
			IngressHosts:              []string{"ingress.host1", "ingress.host2"},
			Image:                     "path/to/Image:latest",
			AdditionalParameters:      []string{"--testParam", "testValue"},
			KubeContext:               "testCluster",
			Namespace:                 "deploymentNamespace",
		}

		dockerConfigJSON := `{"kind": "Secret","data":{".dockerconfigjson": "ThisIsOurBase64EncodedSecret=="}}`

		e := execMockRunner{
			stdoutReturn: map[string]string{
				"kubectl --insecure-skip-tls-verify=true create secret docker-registry regsecret --docker-server=my.registry:55555 --docker-username=registryUser --docker-password=******** --dry-run=true --output=json": dockerConfigJSON,
			},
		}

		runKubernetesDeploy(opts, &e)

		assert.Equal(t, "helm", e.calls[0].exec, "Wrong init command")
		assert.Equal(t, []string{"init", "--client-only"}, e.calls[0].params, "Wrong init parameters")

		assert.Equal(t, "kubectl", e.calls[1].exec, "Wrong secret creation command")
		assert.Equal(t, []string{"--insecure-skip-tls-verify=true", "create", "secret", "docker-registry", "regsecret", "--docker-server=my.registry:55555", "--docker-username=registryUser", "--docker-password=********", "--dry-run=true", "--output=json"}, e.calls[1].params, "Wrong secret creation parameters")

		assert.Equal(t, "helm", e.calls[2].exec, "Wrong upgrade command")
		assert.Equal(t, []string{
			"upgrade",
			"deploymentName",
			"path/to/chart",
			"--install",
			"--force",
			"--namespace",
			"deploymentNamespace",
			"--wait",
			"--timeout",
			"400",
			"--set",
			"image.repository=my.registry:55555/path/to/Image,image.tag=latest,secret.dockerconfigjson=ThisIsOurBase64EncodedSecret==,ingress.hosts[0]=ingress.host1,ingress.hosts[1]=ingress.host2",
			"--kube-context",
			"testCluster",
			"--testParam",
			"testValue",
		}, e.calls[2].params, "Wrong upgrade parameters")
	})
}

func TestSplitRegistryURL(t *testing.T) {
	tt := []struct {
		in          string
		outProtocol string
		outRegistry string
		outError    error
	}{
		{in: "https://my.registry.com", outProtocol: "https", outRegistry: "my.registry.com", outError: nil},
		{in: "https://", outProtocol: "", outRegistry: "", outError: fmt.Errorf("Failed to split registry url 'https://'")},
		{in: "my.registry.com", outProtocol: "", outRegistry: "", outError: fmt.Errorf("Failed to split registry url 'my.registry.com'")},
		{in: "", outProtocol: "", outRegistry: "", outError: fmt.Errorf("Failed to split registry url ''")},
		{in: "https://https://my.registry.com", outProtocol: "", outRegistry: "", outError: fmt.Errorf("Failed to split registry url 'https://https://my.registry.com'")},
	}

	for _, test := range tt {
		p, r, err := splitRegistryURL(test.in)
		assert.Equal(t, test.outProtocol, p, "Protocol value unexpected")
		assert.Equal(t, test.outRegistry, r, "Registry value unexpected")
		assert.Equal(t, test.outError, err, "Error value not as expected")
	}

}

func TestSplitImageName(t *testing.T) {
	tt := []struct {
		in       string
		outImage string
		outTag   string
		outError error
	}{
		{in: "", outImage: "", outTag: "", outError: fmt.Errorf("Failed to split image name ''")},
		{in: "path/to/image", outImage: "path/to/image", outTag: "", outError: nil},
		{in: "path/to/image:tag", outImage: "path/to/image", outTag: "tag", outError: nil},
		{in: "https://my.registry.com/path/to/image:tag", outImage: "", outTag: "", outError: fmt.Errorf("Failed to split image name 'https://my.registry.com/path/to/image:tag'")},
	}
	for _, test := range tt {
		i, tag, err := splitFullImageName(test.in)
		assert.Equal(t, test.outImage, i, "Image value unexpected")
		assert.Equal(t, test.outTag, tag, "Tag value unexpected")
		assert.Equal(t, test.outError, err, "Error value not as expected")
	}
}
