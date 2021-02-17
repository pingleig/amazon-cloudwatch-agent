// Copyright Amazon.com, Inc. or its affiliates. All Rights Reserved.
// SPDX-License-Identifier: MIT

package ecsservicediscovery

import (
	"github.com/aws/aws-sdk-go/aws"
	"log"
)

// Filter out the tasks not matching the discovery configs
// Filter out the tasks with nil task definition
type TaskFilterProcessor struct {
}

func NewTaskFilterProcessor() *TaskFilterProcessor {
	return &TaskFilterProcessor{}
}

func (p *TaskFilterProcessor) Process(cluster string, taskList []*DecoratedTask) ([]*DecoratedTask, error) {
	log.Printf("D! TaskFilter before filter %d", len(taskList))
	var filteredClusterTasks []*DecoratedTask
	for _, v := range taskList {
		if v.ServiceName != "" || v.DockerLabelBased || v.TaskDefinitionBased {
			filteredClusterTasks = append(filteredClusterTasks, v)
			log.Printf("D! TaskFilter includes %s docker label %t", aws.StringValue(v.Task.TaskArn), v.DockerLabelBased)
		}
		log.Printf("D! TaskFilter ingore %s", aws.StringValue(v.Task.TaskArn))
	}
	log.Printf("D! TaskFilter after filter %d", len(filteredClusterTasks))
	return filteredClusterTasks, nil
}

func (p *TaskFilterProcessor) ProcessorName() string {
	return "TaskFilterProcessor"
}
