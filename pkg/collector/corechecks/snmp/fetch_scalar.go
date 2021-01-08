package snmp

import (
	"fmt"
	"github.com/DataDog/datadog-agent/pkg/util/log"
)

func fetchScalarOidsWithBatching(session sessionAPI, oids []string, oidBatchSize int) (scalarResultValuesType, error) {
	retValues := make(scalarResultValuesType)

	batches, err := createStringBatches(oids, oidBatchSize)
	if err != nil {
		return nil, fmt.Errorf("failed to create oid batches: %s", err)
	}

	for _, batchOids := range batches {
		results, err := fetchScalarOids(session, batchOids)
		if err != nil {
			return nil, fmt.Errorf("failed to fetch scalar oids: %s", err.Error())
		}
		for k, v := range results {
			retValues[k] = v
		}
	}
	return retValues, nil
}

func fetchScalarOids(session sessionAPI, oids []string) (scalarResultValuesType, error) {
	// Get results
	log.Debugf("fetchScalarOidsWithBatching() oids: %v", oids)
	results, err := session.Get(oids)
	log.Debugf("fetchColumnOidsWithBatching() results: %v", results)
	if err != nil {
		return nil, fmt.Errorf("error getting oids: %s", err.Error())
	}
	return resultToScalarValues(results), nil
}
