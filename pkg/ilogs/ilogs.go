package ilogs

import (
	"bufio"
	"context"
	"fmt"
	"github.com/manifoldco/promptui"
	log "github.com/sirupsen/logrus"
	"io"
	corev1 "k8s.io/api/core/v1"
	"k8s.io/client-go/kubernetes"
	"k8s.io/client-go/rest"
)

type Config struct {
	Namespace string
	PodFilter string
	ContainerFilter string
	//TailLine *int64
	Naked bool
	VimMode bool
}

type Ilogs struct {
	restConfig *rest.Config
	config *Config
}

func NewIlogs(restConfig *rest.Config, config *Config) *Ilogs {
	log.WithFields(log.Fields{
		"containerFilter": config.ContainerFilter,
		"podFilter": config.PodFilter,
		"Vim Mode": config.VimMode,
		"Naked": config.Naked,
		"Namespace": config.Namespace,
	}).Debug("ilogs config values...")

	return &Ilogs{
		restConfig: restConfig,
		config: config,
	}
}

func (l *Ilogs) selectPod(pods []corev1.Pod) (corev1.Pod, error)  {

	if len(pods) == 1 {
		return pods[0], nil
	}

	templates := podTemplate
	if l.config.Naked {
		templates = podTemplateNaked
	}

	podsPrompt := promptui.Select{
		Label: "Select Pod",
		Items: pods,
		Templates: templates,
		IsVimMode: l.config.VimMode,
	}

	i, _, err := podsPrompt.Run()
	if err != nil {
		return pods[i], err
	}

	return pods[i], nil
}

func (l *Ilogs) selectContainer(containers []corev1.Container) (corev1.Container, error) {
	if len(containers) == 1 {
		return containers[0], nil
	}

	templates := containerTemplates
	if l.config.Naked {
		templates = containerTemplatesNaked
	}

	containersPrompt := promptui.Select{
		Label: "Select Container",
		Items: containers,
		Templates: templates,
		IsVimMode: l.config.VimMode,
	}

	i, _, err := containersPrompt.Run()
	if err != nil {
		return containers[i], err
	}

	return containers[i], err
}

func (l *Ilogs) Do() error {
	client, err := kubernetes.NewForConfig(l.restConfig)
	if err != nil {
		return err
	}

	pods, err := getAllPods(client, l.config.Namespace)
	if err != nil {
		return err
	}

	filteredPods, err := l.matchPods(pods)
	if err != nil {
		return err
	}

	pod, err := l.selectPod(filteredPods.Items)
	if err != nil {
		return err
	}

	containers, err := l.matchContainers(pod)
	if err != nil {
		return err
	}

	container, err := l.selectContainer(containers)
	if err != nil {
		return err
	}

	log.WithFields(log.Fields{
		"pod":       pod.GetName(),
		"container": container.Name,
		"namespace": l.config.Namespace,
	}).Info("logs pod...")

	err = l.Logs(l.restConfig, pod, container)
	if err != nil {
		return err
	}
	return nil
}

func (l *Ilogs) Logs(restConfig *rest.Config, pod corev1.Pod, container corev1.Container) error {
	client, err := kubernetes.NewForConfig(restConfig)
	if err != nil {
		return err
	}
	var line int64 = 200
	namespace := l.config.Namespace
	logOpt := &corev1.PodLogOptions{
		Container: container.Name,
		Follow: true,
		TailLines: &line,
	}

	req := client.CoreV1().Pods(namespace).GetLogs(pod.Name, logOpt)

	log.WithFields(log.Fields{
		"URL": req.URL(),
	}).Debug("Request")

	stream, err := req.Stream(context.TODO())
	if err != nil {
		return err
	}
	defer stream.Close()

	buf := bufio.NewReader(stream)
	for  {
		bytes, err := buf.ReadBytes('\n')
		if err != nil {
			if err != io.EOF {
				return err
			}
			return nil
		}
		fmt.Print(string(bytes))
	}
}