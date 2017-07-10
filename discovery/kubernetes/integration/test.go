package integration

import (
	"context"
	"os"
	"testing"
	"time"

	"k8s.io/apimachinery/pkg/util/yaml"
	"k8s.io/client-go/kubernetes"
	"k8s.io/client-go/pkg/api/v1"
	"k8s.io/client-go/pkg/apis/extensions/v1beta1"
	"k8s.io/client-go/rest"

	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/util/intstr"

	"github.com/docker/libcompose/docker"
	"github.com/docker/libcompose/docker/ctx"
	"github.com/docker/libcompose/project"
	"github.com/docker/libcompose/project/options"
)

type tester struct {
	project project.APIProject
	client  *kubernetes.Clientset
}

// NewIntegrationTester has a TODO documentation
func NewIntegrationTester() *tester {
	return &tester{}
}

func (it *tester) Launch() error {
	project, err := docker.NewProject(&ctx.Context{
		Context: project.Context{
			ComposeFiles: []string{"./kubernetes/integration/fixtures/k8s.yaml"},
			ProjectName:  "k8s-prom-test",
		},
	}, nil)

	if err != nil {
		return err
	}

	err = project.Up(context.Background(), options.Up{})
	if err != nil {
		return err
	}

	it.project = project
	// TODO(gouthamve): Wait for k8s ready not dumb sleep.
	time.Sleep(15 * time.Second)
	cli, err := kubernetes.NewForConfig(&rest.Config{
		Host: "localhost:8080",
	})
	if err != nil {
		return err
	}

	it.client = cli

	namespace := "default"

	// Launch prometheus there with basic config.
	deploy := &v1beta1.Deployment{}
	df, err := os.Open("./kubernetes/integration/fixtures/prometheus.yml")
	if err != nil {
		return err
	}

	if err := yaml.NewYAMLOrJSONDecoder(df, 100).Decode(deploy); err != nil {
		return err
	}

	_, err = it.client.Extensions().Deployments(namespace).Create(deploy)
	if err != nil {
		return err
	}

	service := &v1.Service{
		ObjectMeta: metav1.ObjectMeta{
			Name: "prometheus",
			Labels: map[string]string{
				"app": "prometheus",
			},
		},
		Spec: v1.ServiceSpec{
			Type: v1.ServiceTypeNodePort,
			Ports: []v1.ServicePort{
				v1.ServicePort{
					Name:       "web",
					Port:       9090,
					TargetPort: intstr.FromInt(9090),
				},
			},
			Selector: map[string]string{
				"app": "prometheus",
			},
		},
	}

	_, err = it.client.CoreV1().Services("default").Create(service)
	if err != nil {
		return err
	}
	return nil
}

func (it *tester) Teardown() error {
	return it.project.Down(context.Background(), options.Down{})
}

func (it *tester) Test(t *testing.T) {
	time.Sleep(3 * time.Minute)
}
