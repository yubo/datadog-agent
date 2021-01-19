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
	values, err := doFetchScalarOids(session, oids)
	if err != nil {
		return nil, err
	}
	return values, nil
}

func doFetchScalarOids(session sessionAPI, oids []string) (scalarResultValuesType, error) {
	// Get results
	log.Debugf("fetchScalarOidsWithBatching() oids: %v", oids)
	results, err := session.Get(oids)
	log.Debugf("fetchScalarOidsWithBatching() results: %v", results)
	if err != nil {
		return nil, fmt.Errorf("error getting oids: %s", err.Error())
	}
	values := resultToScalarValues(results)
	retryScalarOids(session, results, values)
	return values, nil
}

// retryScalarOids retries on NoSuchObject or NoSuchInstance for scalar oids not ending with `.0`.
// This helps keeping compatibility with python implementation.
// This is not need in normal circumstances where scalar OIDs end with `.0`.
// If the oid does not end with `.0`, we will retry by appending `.0` to it.
func retryScalarOids(session sessionAPI, results *gosnmp.SnmpPacket, valuesToUpdate scalarResultValuesType) {
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
					valuesToUpdate[initialOid] = value
				}
			}
		}
	}
}
