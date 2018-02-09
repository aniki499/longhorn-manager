package controller

import (
	"os"
	"time"

	"github.com/Sirupsen/logrus"
	"github.com/pkg/errors"

	corev1 "k8s.io/api/core/v1"
	"k8s.io/client-go/informers"
	clientset "k8s.io/client-go/kubernetes"

	"github.com/rancher/longhorn-manager/k8s"
	longhorn "github.com/rancher/longhorn-manager/k8s/pkg/apis/longhorn/v1alpha1"
	lhclientset "github.com/rancher/longhorn-manager/k8s/pkg/client/clientset/versioned"
	lhinformers "github.com/rancher/longhorn-manager/k8s/pkg/client/informers/externalversions"
)

var (
	Workers              = 5
	longhornFinalizerKey = longhorn.SchemeGroupVersion.Group
)

func StartControllers(controllerID string) error {
	namespace := os.Getenv(k8s.EnvPodNamespace)
	if namespace == "" {
		logrus.Warnf("Cannot detect pod namespace, environment variable %v is missing, " +
			"using default namespace")
		namespace = corev1.NamespaceDefault
	}

	config, err := k8s.GetClientConfig("")
	if err != nil {
		return errors.Wrapf(err, "unable to get client config")
	}

	kubeClient, err := clientset.NewForConfig(config)
	if err != nil {
		return errors.Wrapf(err, "unable to get k8s client")
	}

	lhClient, err := lhclientset.NewForConfig(config)
	if err != nil {
		return errors.Wrapf(err, "unable to get clientset")
	}

	kubeInformerFactory := informers.NewSharedInformerFactory(kubeClient, time.Second*30)
	lhInformerFactory := lhinformers.NewSharedInformerFactory(lhClient, time.Second*30)

	replicaInformer := lhInformerFactory.Longhorn().V1alpha1().Replicas()
	podInformer := kubeInformerFactory.Core().V1().Pods()
	jobInformer := kubeInformerFactory.Batch().V1().Jobs()

	rc := NewReplicaController(replicaInformer, podInformer, jobInformer, lhClient, kubeClient, namespace, controllerID)

	//FIXME stopch should be exposed
	stopCh := make(chan struct{})
	go kubeInformerFactory.Start(stopCh)
	go lhInformerFactory.Start(stopCh)
	go rc.Run(Workers, stopCh)

	return nil
}
