// Copyright Amazon.com, Inc. or its affiliates. All Rights Reserved.
// SPDX-License-Identifier: MIT

package extractors

import (
	"github.com/aws/amazon-cloudwatch-agent/internal/containerinsightscommon"
	cinfo "github.com/google/cadvisor/info/v1"
)

// aggregate fileds
func aggregate(fields []map[string]float64) map[string]float64 {
	if len(fields) == 0 {
		return nil
	}

	result := make(map[string]float64)
	// Use the first element as the base
	for k, v := range fields[0] {
		result[k] = v
	}

	if len(fields) == 1 {
		return result
	}

	for i := 1; i < len(fields); i++ {
		for k, v := range result {
			result[k] = v + fields[i][k]
		}
	}
	return result
}

func GetStats(info *cinfo.ContainerInfo) *cinfo.ContainerStats {
	if len(info.Stats) == 0 {
		return nil
	}
	// When there is more than one stats point, always use the last one
	return info.Stats[len(info.Stats)-1]
}

func isInfraContainer(info *cinfo.ContainerInfo, containerType string) bool {
	if containerType != containerinsightscommon.TypeContainer {
		return false
	}

	// Docker
	switch info.Spec.Labels[containerNameLabel] {
	case infraContainerName:
		return true // docker
	case "":
		return true // containerd
	default:
		return false
	}
}
