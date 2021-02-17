// Copyright Amazon.com, Inc. or its affiliates. All Rights Reserved.
// SPDX-License-Identifier: MIT

package ecsservicediscovery

import (
	"github.com/aws/aws-sdk-go/aws"
	"log"
)

// Tag the Tasks that matched the Docker Label based SD Discovery
type DockerLabelDiscoveryProcessor struct {
	label string
}

func NewDockerLabelDiscoveryProcessor(d *DockerLabelConfig) *DockerLabelDiscoveryProcessor {
	if d == nil {
		log.Printf("W! DockerLabel there is no docker config")
		return &DockerLabelDiscoveryProcessor{label: ""}
	}
	log.Printf("D! DockerLabel is using port label %q", d.PortLabel)
	return &DockerLabelDiscoveryProcessor{label: d.PortLabel}
}

func (p *DockerLabelDiscoveryProcessor) Process(cluster string, taskList []*DecoratedTask) ([]*DecoratedTask, error) {
	log.Printf("D! DockerLabel got %d tasks", len(taskList))
	if p.label == "" {
		log.Printf("W! DockerLabel has empty label")
		return taskList, nil
	}

	for _, v := range taskList {
		log.Printf("D! DockerLabel checking task %s with %d cotnainers", aws.StringValue(v.Task.TaskArn),
			len(v.TaskDefinition.ContainerDefinitions))
		for _, d := range v.TaskDefinition.ContainerDefinitions {
			log.Printf("D! Container %s labels %v", aws.StringValue(d.Name), m2m(d.DockerLabels))
			if _, ok := d.DockerLabels[p.label]; ok {
				v.DockerLabelBased = true
				log.Printf("D! DockerLabel matched task %s", aws.StringValue(v.Task.TaskArn))
				break
			}
		}
	}
	return taskList, nil
}

func (p *DockerLabelDiscoveryProcessor) ProcessorName() string {
	return "DockerLabelDiscoveryProcessor"
}

func m2m(mp map[string]*string) map[string]string {
	m := make(map[string]string)
	for k, v := range mp {
		m[k] = aws.StringValue(v)
	}
	return m
}
