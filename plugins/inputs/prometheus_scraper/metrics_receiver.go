// Copyright Amazon.com, Inc. or its affiliates. All Rights Reserved.
// SPDX-License-Identifier: MIT

package prometheus_scraper

import (
	"errors"
	"log"
	"math"

	"github.com/prometheus/common/model"
	"github.com/prometheus/prometheus/pkg/labels"
	"github.com/prometheus/prometheus/pkg/value"
	"github.com/prometheus/prometheus/storage"
)

type PrometheusMetricBatch []*PrometheusMetric

type PrometheusMetric struct {
	tags       map[string]string
	metricName string
	// We use this name to look up metric type because user can relabel __name___.
	// See https://github.com/aws/amazon-cloudwatch-agent/issues/190
	metricNameBeforeRelabel string
	jobBeforeRelabel        string
	instanceBeforeRelabel   string
	metricValue             float64
	metricType              string
	timeInMS                int64 // Unix time in milli-seconds
}

func (pm *PrometheusMetric) isValueValid() bool {
	//treat NaN and +/-Inf values as invalid as emf log doesn't support them
	return !value.IsStaleNaN(pm.metricValue) && !math.IsNaN(pm.metricValue) && !math.IsInf(pm.metricValue, 0)
}

// metricsReceiver implement interface Appender for prometheus scarper to append metrics
type metricsReceiver struct {
	pmbCh chan<- PrometheusMetricBatch
}

type metricAppender struct {
	receiver *metricsReceiver
	batch    PrometheusMetricBatch
}

func (mr *metricsReceiver) Appender() storage.Appender {
	return &metricAppender{receiver: mr, batch: PrometheusMetricBatch{}}
}

func (mr *metricsReceiver) feed(batch PrometheusMetricBatch) error {
	select {
	case mr.pmbCh <- batch:
	default:
		log.Println("W! metric batch drop due to channel full")
	}
	return nil
}

func (ma *metricAppender) Add(ls labels.Labels, t int64, v float64) (uint64, error) {
	metricName := ""

	labelMap := make(map[string]string, len(ls))
	for _, l := range ls {
		if l.Name == model.MetricNameLabel {
			metricName = l.Value
			continue
		}
		labelMap[l.Name] = l.Value
	}

	if metricName == "" {
		// The error should never happen, print log here for debugging
		log.Println("E! receive invalid prometheus metric, metricName is missing")
		return uint64(0), errors.New("metricName of the times-series is missing")
	}

	pm := &PrometheusMetric{
		metricName:              metricName,
		metricNameBeforeRelabel: ls.Get(savedScrapeNameLabel),
		jobBeforeRelabel:        ls.Get(savedScrapeJobLabel),
		instanceBeforeRelabel:   ls.Get(savedScrapeInstanceLabel),
		metricValue:             v,
		timeInMS:                t,
	}
	log.Printf("%s %v", pm.metricName, labelMap)

	// Remove magic labels
	delete(labelMap, savedScrapeNameLabel)
	delete(labelMap, savedScrapeJobLabel)
	delete(labelMap, savedScrapeInstanceLabel)

	pm.tags = labelMap
	ma.batch = append(ma.batch, pm)
	return uint64(0), nil //return 0 to indicate caching is not supported
}

// always returns error since caching is not supported by Add() function
func (ma *metricAppender) AddFast(_ uint64, _ int64, _ float64) error {
	return storage.ErrNotFound
}

func (ma *metricAppender) Commit() error {
	return ma.receiver.feed(ma.batch)
}

func (ma *metricAppender) Rollback() error {
	// wipe the batch
	ma.batch = PrometheusMetricBatch{}
	return nil
}
