package snmp

import (
	"fmt"
	"github.com/DataDog/datadog-agent/pkg/util/log"
	"github.com/soniah/gosnmp"
	"strings"
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
	log.Debugf("fetchScalarOidsWithBatching() results: %v", results)
	if err != nil {
		return nil, fmt.Errorf("error getting oids: %s", err.Error())
	}
	values := resultToScalarValues(results)

	// Retry on NoSuchObject or NoSuchInstance for scalar oids not ending with .0
	// This helps keeping compatibility with python implementation.
	retryOids := make(map[string]string)
	for _, variable := range results.Variables {
		oid := variable.Name
		if (variable.Type == gosnmp.NoSuchObject || variable.Type == gosnmp.NoSuchInstance) && !strings.HasSuffix(".0", oid) {
			retryOids[oid] = oid + ".0"
		}
	}
	if len(retryOids) > 0 {
		fetchOids := make([]string, 0, len(retryOids))
		for _, oid := range retryOids {
			fetchOids = append(fetchOids, oid)
		}
		retryResults, err := session.Get(fetchOids)
		if err != nil {
			log.Debugf("failed to oids `%v` on retry: %v", retryOids, err)
		} else {
			retryValues := resultToScalarValues(retryResults)
			for initialOid, actualOid := range retryOids {
				if value, ok := retryValues[actualOid]; ok {
					values[initialOid] = value
				}
			}
		}
	}
	return values, nil
}
