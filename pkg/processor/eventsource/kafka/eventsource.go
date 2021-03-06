/*
Copyright 2017 The Nuclio Authors.

Licensed under the Apache License, Version 2.0 (the "License");
you may not use this file except in compliance with the License.
You may obtain a copy of the License at

    http://www.apache.org/licenses/LICENSE-2.0

Unless required by applicable law or agreed to in writing, software
distributed under the License is distributed on an "AS IS" BASIS,
WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
See the License for the specific language governing permissions and
limitations under the License.
*/

package kafka

import (
	"github.com/nuclio/nuclio/pkg/common"
	"github.com/nuclio/nuclio/pkg/errors"
	"github.com/nuclio/nuclio/pkg/processor/eventsource"
	"github.com/nuclio/nuclio/pkg/processor/worker"

	"github.com/Shopify/sarama"
	"github.com/nuclio/nuclio-sdk"
)

type kafka struct {
	eventsource.AbstractEventSource
	event         Event
	configuration *Configuration
	worker        *worker.Worker
	consumer      sarama.Consumer
	partitions    []*partition
}

func newEventSource(parentLogger nuclio.Logger,
	workerAllocator worker.Allocator,
	configuration *Configuration) (eventsource.EventSource, error) {
	var err error

	newEventSource := &kafka{
		AbstractEventSource: eventsource.AbstractEventSource{
			ID:              configuration.ID,
			Logger:          parentLogger.GetChild(configuration.ID).(nuclio.Logger),
			WorkerAllocator: workerAllocator,
			Class:           "async",
			Kind:            "kafka",
		},
		configuration: configuration,
	}

	newEventSource.consumer, err = sarama.NewConsumer([]string{configuration.Host}, nil)
	if err != nil {
		return nil, errors.Wrap(err, "Failed to create consumer")
	}

	// iterate over partitions and create
	for _, partitionID := range configuration.Partitions {

		// create the partition
		partition, err := newPartition(newEventSource.Logger, newEventSource, partitionID)
		if err != nil {
			return nil, errors.Wrap(err, "Failed to create partition")
		}

		// add partition
		newEventSource.partitions = append(newEventSource.partitions, partition)
	}

	return newEventSource, nil
}

func (k *kafka) Start(checkpoint eventsource.Checkpoint) error {
	k.Logger.InfoWith("Starting",
		"streamName", k.configuration.Host,
		"topic", k.configuration.Topic)

	for _, partitionInstance := range k.partitions {

		// start reading from partition
		go func(partitionInstance *partition) {
			if err := partitionInstance.readFromPartition(); err != nil {
				k.Logger.ErrorWith("Failed to read from partition", "err", err)
			}
		}(partitionInstance)
	}

	return nil
}

func (k *kafka) Stop(force bool) (eventsource.Checkpoint, error) {

	// TODO
	return nil, nil
}

func (k *kafka) GetConfig() map[string]interface{} {
	return common.StructureToMap(k.configuration)
}
