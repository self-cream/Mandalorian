package mandalorian

import (
	"github.com/NJUPT-ISL/Mandalorian/pkg/scheduler/framework"
	"k8s.io/apimachinery/pkg/runtime"
	scv "github.com/NJUPT-ISL/SCV/api/v1"
	"k8s.io/client-go/tools/clientcmd"
	"k8s.io/klog/v2"
	"sigs.k8s.io/controller-runtime/pkg/client"
)

const (
	Name = "Mandalorian"
	ScoreWeight = 2
)


var _ framework.FilterPlugin = &Mandalorian{}
var _ framework.ScorePlugin = &Mandalorian{}

var scheme = runtime.NewScheme()

type Mandalorian struct {
	handle    framework.Handle
	scvClient client.Client
}

func (m *Mandalorian) Name() string {
	return Name
}

func New(_ runtime.Object, h framework.Handle) (framework.Plugin, error) {
	return &Mandalorian{
		handle: h,
		scvClient: NewScvClient(),
	}, nil
}

func NewScvClient() client.Client {
	err := scv.AddToScheme(scheme)
	if err != nil {
		klog.Errorf("Add SCV CRD to Scheme Error: %v", err)
		return nil
	}
	config, err := clientcmd.BuildConfigFromFlags("", "")
	if err != nil {
		klog.Errorf("Get Kubernetes Config Error: %v", err)
		return nil
	}
	c, err := client.New(config, client.Options{
		Scheme: scheme,
	})
	if err != nil {
		klog.Errorf("New Client Error: %v", err)
		return nil
	}
	return c
}