// Unless explicitly stated otherwise all files in this repository are licensed
// under the Apache License Version 2.0.
// This product includes software developed at Datadog (https://www.datadoghq.com/).
// Copyright 2016-present Datadog, Inc.

// +build containerd

package collectors

import (
	"context"
	"io"
	"strings"

	"github.com/DataDog/datadog-agent/pkg/config"
	"github.com/DataDog/datadog-agent/pkg/tagger/utils"
	"github.com/DataDog/datadog-agent/pkg/util/containers"
	"github.com/DataDog/datadog-agent/pkg/util/log"
	"github.com/containerd/containerd"
	api "github.com/containerd/containerd/api/events"
	"github.com/containerd/containerd/events"
	"github.com/containerd/containerd/namespaces"
	_ "github.com/containerd/cri/pkg/store/container"
	"github.com/containerd/typeurl"
	"github.com/gobwas/glob"
	"k8s.io/kubernetes/pkg/kubelet/types"

	"github.com/DataDog/datadog-agent/pkg/errors"
	"github.com/DataDog/datadog-agent/pkg/status/health"
	containerdutil "github.com/DataDog/datadog-agent/pkg/util/containerd"
)

const (
	containerdCollectorName = "containerd"
)

// ContainerdCollector listens to events on the containerd socket to get new/dead containers
// and feed a stram of TagInfo. It requires access to the containerd socket.
// It will also embed DockerExtractor collectors for container tagging.
type ContainerdCollector struct {
	containerdUtil containerdutil.ContainerdItf
	cancelFunc     context.CancelFunc
	infoOut        chan<- []*TagInfo
	labelsAsTags   map[string]string
	envAsTags      map[string]string
	globLabels     map[string]glob.Glob
}

// Detect tries to connect to the containerd socket and returns success
func (c *ContainerdCollector) Detect(out chan<- []*TagInfo) (CollectionMode, error) {
	cu, err := containerdutil.GetContainerdUtil()
	if err != nil {
		return NoCollection, err
	}

	c.containerdUtil = cu
	c.infoOut = out

	c.labelsAsTags = config.Datadog.GetStringMapString("containerd_labels_as_tags")
	c.envAsTags = config.Datadog.GetStringMapString("containerd_annotations_as_tags")

	return StreamCollection, nil
}

// Stream runs the continuous event watching loop and sends new info
// to the channel. But be called in a goroutine.
func (c *ContainerdCollector) Stream() error {
	healthHandle := health.RegisterLiveness("tagger-containerd")

	ctx, cancel := context.WithCancel(context.Background())
	c.cancelFunc = cancel

	eventService := c.containerdUtil.GetEvents()
	ctxNamespace := namespaces.WithNamespace(ctx, c.containerdUtil.Namespace())
	events, errs := eventService.Subscribe(ctxNamespace)

	for {
		select {
		case <-ctx.Done():
			healthHandle.Deregister() //nolint:errcheck
			return nil
		case <-healthHandle.C:
		case evt := <-events:
			c.processEvent(evt)
		case err := <-errs:
			if err != nil && err != io.EOF {
				log.Errorf("stopping collection: %s", err)
				return err
			}
			return nil
		}
	}
}

// Stop queues a shutdown of DockerListener
func (c *ContainerdCollector) Stop() error {
	if c.cancelFunc != nil {
		c.cancelFunc()
	}
	return nil
}

// Fetch inspect a given container to get its tags on-demand (cache miss)
func (c *ContainerdCollector) Fetch(entity string) ([]string, []string, []string, error) {
	entityType, cID := containers.SplitEntityName(entity)
	if entityType != containers.ContainerEntityName || len(cID) == 0 {
		return nil, nil, nil, nil
	}
	low, orchestrator, high, _, err := c.fetchForContainerdID(cID)
	return low, orchestrator, high, err
}

func (c *ContainerdCollector) processEvent(e *events.Envelope) {
	var info *TagInfo

	switch e.Topic {
	case "/containers/delete":
		ev, err := typeurl.UnmarshalAny(e.Event)
		if err != nil {
			log.Debug("Failed to parse containerd delete event: %s", err)
			return
		}

		containerDelete, ok := ev.(*api.ContainerDelete)
		if !ok {
			log.Debugf("Failed to unmarshal containerd delete event: %s", err)
			return
		}

		info = &TagInfo{
			Entity:       containers.BuildEntityName(containers.RuntimeNameContainerd, containerDelete.ID),
			Source:       containerdCollectorName,
			DeleteEntity: true,
		}
	case "/containers/create":
		ev, err := typeurl.UnmarshalAny(e.Event)
		if err != nil {
			log.Debug("Failed to parse containerd create event: %s", err)
			return
		}

		containerCreate, ok := ev.(*api.ContainerCreate)
		if !ok {
			log.Debugf("Failed to unmarshal containerd create event: %s", err)
			return
		}

		low, orchestrator, high, standard, err := c.fetchForContainerdID(containerCreate.ID)
		if err != nil {
			log.Debugf("Error fetching tags for container '%s': %v", containerCreate.ID, err)
			return
		}

		info = &TagInfo{
			Entity:               containers.ContainerEntityName,
			Source:               containerdCollectorName,
			LowCardTags:          low,
			OrchestratorCardTags: orchestrator,
			HighCardTags:         high,
			StandardTags:         standard,
		}
	default:
		return // Nothing to see here
	}
	c.infoOut <- []*TagInfo{info}
}

func (c *ContainerdCollector) fetchForContainerdID(cID string) ([]string, []string, []string, []string, error) {
	container, err := c.containerdUtil.LoadContainer(cID)
	if err != nil {
		if !errors.IsNotFound(err) {
			log.Debugf("Failed to inspect container %s - %s", cID, err)
		}
		return nil, nil, nil, nil, err
	}
	low, orchestrator, high, standard := c.extractFromInspect(container)
	return low, orchestrator, high, standard, nil
}

// extractFromInspect extract tags for a container inspect JSON
func (c *ContainerdCollector) extractFromInspect(container containerd.Container) ([]string, []string, []string, []string) {
	tags := utils.NewTagList()

	info, err := c.containerdUtil.Info(container)
	if err != nil {
		log.Debugf("Failed to retrieve info for container %s", container.ID())
	} else {
		c.containerdExtractImage(tags, info.Image)
		c.containerdExtractLabels(tags, info.Labels, c.labelsAsTags)
	}

	spec, err := c.containerdUtil.Spec(container)
	if err != nil {
		log.Debugf("Failed to retrieve spec for container %s", container.ID())
	} else {
		c.containerdExtractEnvironmentVariables(tags, spec.Process.Env, c.envAsTags)
	}

	tags.AddHigh("container_id", container.ID())

	low, orchestrator, high, standard := tags.Compute()
	return low, orchestrator, high, standard
}

func (c *ContainerdCollector) containerdExtractImage(tags *utils.TagList, containerImage string) {
	tags.AddLow("image", containerImage)
	imageName, shortImage, imageTag, err := containers.SplitImageName(containerImage)
	if err != nil {
		log.Debugf("Cannot split %s: %s", containerImage, err)
		return
	}
	tags.AddLow("image_name", imageName)
	tags.AddLow("short_image", shortImage)
	tags.AddLow("image_tag", imageTag)
}

func (c *ContainerdCollector) containerdExtractLabels(tags *utils.TagList, labels, labelsAsTags map[string]string) {
	for name, value := range labels {
		switch name {
		case types.KubernetesPodNameLabel:
			tags.AddOrchestrator("pod_name", value)
		case types.KubernetesPodNamespaceLabel:
			tags.AddLow("kube_namespace", value)
		case types.KubernetesContainerNameLabel:
			tags.AddLow("container_name", value)
		default:
			utils.AddMetadataAsTags(name, value, c.labelsAsTags, c.globLabels, tags)
		}
	}
}

func (c *ContainerdCollector) containerdExtractEnvironmentVariables(tags *utils.TagList, envVars []string, envAsTags map[string]string) {
	for _, envEntry := range envVars {
		envSplit := strings.SplitN(envEntry, "=", 2)
		if len(envSplit) != 2 {
			continue
		}
		envName := envSplit[0]
		envValue := envSplit[1]

		switch envName {
		// Standard tags
		case envVarEnv:
			tags.AddStandard(tagKeyEnv, envValue)
		case envVarVersion:
			tags.AddStandard(tagKeyVersion, envValue)
		case envVarService:
			tags.AddStandard(tagKeyService, envValue)
		default:
			if tagName, found := envAsTags[strings.ToLower(envSplit[0])]; found {
				tags.AddAuto(tagName, envValue)
			}
		}
	}
}

func containerdFactory() Collector {
	return &ContainerdCollector{}
}

func init() {
	registerCollector(containerdCollectorName, containerdFactory, NodeRuntime)
}
