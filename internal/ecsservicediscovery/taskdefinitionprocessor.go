// Copyright Amazon.com, Inc. or its affiliates. All Rights Reserved.
// SPDX-License-Identifier: MIT

package ecsservicediscovery

import (
	"github.com/aws/aws-sdk-go/aws"
	"github.com/aws/aws-sdk-go/service/ecs"
	"github.com/hashicorp/golang-lru/simplelru"
	"log"
)

const (
	// ECS Service Quota: https://docs.aws.amazon.com/AmazonECS/latest/developerguide/service-quotas.html
	taskDefCacheSize = 2000
)

// Decorate the tasks with the ECS task definition
type TaskDefinitionProcessor struct {
	svcEcs *ecs.ECS
	stats  *ProcessorStats

	taskDefCache *simplelru.LRU
}

func NewTaskDefinitionProcessor(ecs *ecs.ECS, s *ProcessorStats) *TaskDefinitionProcessor {
	p := &TaskDefinitionProcessor{
		svcEcs: ecs,
		stats:  s,
	}

	// initiate the caching
	lru, err := simplelru.NewLRU(taskDefCacheSize, nil)
	if err != nil {
		panic(err)
	}
	p.taskDefCache = lru
	return p
}

// Process fetches task definition based on task list
func (p *TaskDefinitionProcessor) Process(cluster string, taskList []*DecoratedTask) ([]*DecoratedTask, error) {
	defer func() {
		p.stats.AddStatsCount(LRUCacheSizeTaskDefinition, p.taskDefCache.Len())
	}()

	arn2Definition := make(map[string]*ecs.TaskDefinition)
	for _, t := range taskList {
		arn2Definition[aws.StringValue(t.Task.TaskDefinitionArn)] = nil
	}
	log.Printf("D! TaskDefinitionProcessor found %d task definitions", len(arn2Definition))

	for k, _ := range arn2Definition {
		if k == "" {
			continue
		}

		var td *ecs.TaskDefinition
		if res, ok := p.taskDefCache.Get(k); ok {
			p.stats.AddStats(LRUCacheGetTaskDefinition)
			td = res.(*ecs.TaskDefinition)
		} else {
			resp, err := p.svcEcs.DescribeTaskDefinition(&ecs.DescribeTaskDefinitionInput{TaskDefinition: &k})
			p.stats.AddStats(AWSCLIDescribeTaskDefinition)
			if err != nil {
				return taskList, newServiceDiscoveryError("Failed to describe task definition for "+k, &err)
			}
			p.taskDefCache.Add(k, resp.TaskDefinition)
			td = resp.TaskDefinition
		}
		arn2Definition[k] = td
	}

	for _, v := range taskList {
		v.TaskDefinition = arn2Definition[aws.StringValue(v.Task.TaskDefinitionArn)]
	}
	log.Printf("D! TaskDefinitionProcessor before filter %d tasks", len(taskList))
	taskList = filterNilTaskDefinitionTasks(taskList)
	log.Printf("D! TaskDefinitionProcessor after filter %d tasks", len(taskList))
	return taskList, nil
}

func filterNilTaskDefinitionTasks(taskList []*DecoratedTask) []*DecoratedTask {
	var filteredTasks []*DecoratedTask
	for _, v := range taskList {
		if v.TaskDefinition != nil {
			filteredTasks = append(filteredTasks, v)
		} else {
			log.Printf("D! Task has nil definition arn %s", aws.StringValue(v.Task.TaskArn))
		}
	}
	return filteredTasks
}

func (p *TaskDefinitionProcessor) ProcessorName() string {
	return "TaskDefinitionProcessor"
}
