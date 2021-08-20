package ilogs

import (
	"context"
	"fmt"
	log "github.com/sirupsen/logrus"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/client-go/kubernetes"
	"sort"
	"strings"
)

func getAllPods(client *kubernetes.Clientset, namespace string) (*corev1.PodList, error) {
	listOpt := metav1.ListOptions{}
	pods, err := client.CoreV1().Pods(namespace).List(context.TODO(), listOpt)
	if err != nil {
		return pods, err
	}

	log.WithFields(log.Fields{
		"pods": len(pods.Items),
		"namespaces": namespace,
	}).Debug("total pods discovered")

	return pods, nil
}

func (l *Ilogs) matchPods(pods *corev1.PodList) (corev1.PodList, error) {
	var result corev1.PodList

	log.WithFields(log.Fields{
		"SearchFilter": l.config.PodFilter,
	}).Infof("Get all pods for podFilter...")

	for i, pod := range pods.Items {
		if strings.Contains(pod.GetName(), l.config.PodFilter) {
			result.Items = append(result.Items, pod)
			log.WithFields(log.Fields{
				"PodName": pod.GetName(),
				"index":   i,
			}).Infof("Found pod...")
		}
	}

	if len(result.Items) == 0 {
		err := fmt.Errorf("no pods found for filter: %s", l.config.PodFilter)
		return result, err
	}

	return result, nil
}

func (l *Ilogs) matchContainers(pod corev1.Pod) ([]corev1.Container, error) {
	if l.config.ContainerFilter == "" {
		return pod.Spec.Containers, nil
	}
	log.WithFields(log.Fields{
		"SearchFilter": l.config.ContainerFilter,
	}).Infof("Get all containers for containerFilter...")

	var matchingContainer []corev1.Container
	for i, container := range pod.Spec.Containers {
		if strings.Contains(container.Name, l.config.ContainerFilter) {
			matchingContainer = append(matchingContainer, container)
			log.WithFields(log.Fields{
				"ContainerName": container.Name,
				"index":         i,
			}).Infof("Found container...")
		}
	}

	if len(matchingContainer) == 0 {
		err := fmt.Errorf("no containers found for filter: %s", l.config.ContainerFilter)
		return nil, err
	}

	sort.Slice(matchingContainer, func(i, j int) bool {
		return matchingContainer[i].Name < matchingContainer[j].Name
	})

	return matchingContainer, nil
}
