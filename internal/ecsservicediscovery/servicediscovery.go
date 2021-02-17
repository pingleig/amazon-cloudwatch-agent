// Copyright Amazon.com, Inc. or its affiliates. All Rights Reserved.
// SPDX-License-Identifier: MIT

package ecsservicediscovery

import (
	"log"
	"sync"
	"time"

	internalaws "github.com/aws/amazon-cloudwatch-agent/cfg/aws"
	"github.com/aws/aws-sdk-go/aws"
	"github.com/aws/aws-sdk-go/service/ec2"
	"github.com/aws/aws-sdk-go/service/ecs"
)

type ServiceDiscovery struct {
	Config *ServiceDiscoveryConfig

	svcEcs *ecs.ECS
	svcEc2 *ec2.EC2

	stats             ProcessorStats
	clusterProcessors []Processor
}

func (sd *ServiceDiscovery) init() {
	credentialConfig := &internalaws.CredentialConfig{
		Region: sd.Config.TargetClusterRegion,
	}
	configProvider := credentialConfig.Credentials()
	log.Printf("SD cluster is %s region is %s", sd.Config.TargetCluster, sd.Config.TargetClusterRegion)
	sd.svcEcs = ecs.New(configProvider, aws.NewConfig().WithRegion(sd.Config.TargetClusterRegion).WithMaxRetries(AwsSdkLevelRetryCount))
	sd.svcEc2 = ec2.New(configProvider, aws.NewConfig().WithRegion(sd.Config.TargetClusterRegion).WithMaxRetries(AwsSdkLevelRetryCount))

	sd.initClusterProcessorPipeline()
}

func (sd *ServiceDiscovery) initClusterProcessorPipeline() {
	sd.clusterProcessors = append(sd.clusterProcessors,
		NewTaskProcessor(sd.svcEcs, &sd.stats),
		NewTaskDefinitionProcessor(sd.svcEcs, &sd.stats),
		NewServiceEndpointDiscoveryProcessor(sd.svcEcs, sd.Config.ServiceNamesForTasks, &sd.stats),
		NewDockerLabelDiscoveryProcessor(sd.Config.DockerLabel),
		NewTaskDefinitionDiscoveryProcessor(sd.Config.TaskDefinitions),
		NewTaskFilterProcessor(),
		NewContainerInstanceProcessor(sd.svcEcs, sd.svcEc2, &sd.stats),
		NewTargetsExportProcessor(sd.Config, &sd.stats),
	)
}

func StartECSServiceDiscovery(sd *ServiceDiscovery, shutDownChan chan interface{}, wg *sync.WaitGroup) {
	defer wg.Done()

	if !sd.validateConfig() {
		return
	}

	frequency, _ := time.ParseDuration(sd.Config.Frequency)
	sd.init()
	t := time.NewTicker(frequency)
	defer t.Stop()
	for {
		select {
		case <-shutDownChan:
			return
		case <-t.C:
			sd.work()
		}
	}
}

func (sd *ServiceDiscovery) work() {
	sd.stats.ResetStats()
	var err error
	var clusterTasks []*DecoratedTask
	for _, p := range sd.clusterProcessors {
		clusterTasks, err = p.Process(sd.Config.TargetCluster, clusterTasks)
		// Ignore partial result to avoid overwriting existing targets
		if err != nil {
			log.Printf("E! ECS SD processor: %v got error: %v \n", p.ProcessorName(), err.Error())
			return
		}
	}
	sd.stats.ShowStats()
}

func (sd *ServiceDiscovery) validateConfig() bool {
	if sd.Config == nil {
		return false
	}

	if sd.Config.DockerLabel == nil && len(sd.Config.TaskDefinitions) == 0 && len(sd.Config.ServiceNamesForTasks) == 0 {
		log.Printf("E! Neither docker label based discovery, nor task definition based discovery, nor service name based discovery is enabled.\n")
		return false
	}

	if sd.Config.TargetCluster == "" || sd.Config.TargetClusterRegion == "" {
		log.Printf("E! Target ECS cluster info is not correct.\n")
		return false
	}

	_, err := time.ParseDuration(sd.Config.Frequency)
	if err != nil {
		log.Printf("E! Invalid ECS service discovery frequency: %v.\n", sd.Config.Frequency)
		return false
	}

	return true
}
