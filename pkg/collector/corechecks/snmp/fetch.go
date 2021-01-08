package snmp

import (
	"fmt"
	"github.com/DataDog/datadog-agent/pkg/util/log"
	"sort"
)

func fetchValues(session sessionAPI, config snmpConfig) (*snmpValues, error) {
	scalarResults, err := fetchScalarOidsWithBatching(session, config.OidConfig.scalarOids, config.OidBatchSize)
	if err != nil {
		return &snmpValues{}, fmt.Errorf("failed to fetch scalar oids with batching: %v", err)
	}

	oids := make(map[string]string)
	for _, value := range config.OidConfig.columnOids {
		oids[value] = value
	}
	columnResults, err := fetchColumnOidsWithBatching(session, oids, config.OidBatchSize)
	if err != nil {
		return &snmpValues{}, fmt.Errorf("failed to fetch oids with batching: %v", err)
	}

	return &snmpValues{scalarResults, columnResults}, nil
}

func fetchScalarOidsWithBatching(session sessionAPI, oids []string, oidBatchSize int) (map[string]snmpValue, error) {
	retValues := make(map[string]snmpValue)

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

func fetchScalarOids(session sessionAPI, oids []string) (map[string]snmpValue, error) {
	// Get results
	log.Debugf("fetchScalarOidsWithBatching() oids: %v", oids)
	results, err := session.Get(oids)
	log.Debugf("fetchColumnOidsWithBatching() results: %v", results)
	if err != nil {
		return nil, fmt.Errorf("error getting oids: %s", err.Error())
	}
	return resultToScalarValues(results), nil
}

func fetchColumnOidsWithBatching(session sessionAPI, oids map[string]string, oidBatchSize int) (map[string]map[string]snmpValue, error) {
	// Get results
	// TODO: Improve batching algorithm and make it more readable
	retValues := make(map[string]map[string]snmpValue)

	columnOids := getOidsMapKeys(oids)
	batches, err := createStringBatches(columnOids, oidBatchSize)
	if err != nil {
		return nil, fmt.Errorf("failed to create column oid batches: %s", err)
	}

	for _, batchColumnOids := range batches {
		oidsToFetch := make(map[string]string)
		for _, oid := range batchColumnOids {
			oidsToFetch[oid] = oids[oid]
		}

		results, err := fetchColumnOids(session, oidsToFetch)
		if err != nil {
			return nil, fmt.Errorf("failed to fetch column oids: %s", err)
		}

		for columnOid, instanceOids := range results {
			if _, ok := retValues[columnOid]; !ok {
				retValues[columnOid] = instanceOids
				continue
			}
			for oid, value := range instanceOids {
				retValues[columnOid][oid] = value
			}
		}
	}
	return retValues, nil
}

// fetchColumnOids has an `oids` argument representing a `map[string]string`,
// the key of the map is the column oid, and the value is the oid used to fetch the next value for the column.
// The value oid might be equal to column oid or a row oid of the same column.
//
// Return value structure:  map[columnOID]map[index]interface(float64 or string)
func fetchColumnOids(session sessionAPI, oids map[string]string) (map[string]map[string]snmpValue, error) {
	returnValues := make(map[string]map[string]snmpValue)
	curOids := oids
	for {
		log.Debugf("fetchColumnOidsWithBatching() curOids  : %v", curOids)
		if len(curOids) == 0 {
			break
		}
		var columnOids, bulkOids []string
		for k, v := range curOids {
			columnOids = append(columnOids, k)
			bulkOids = append(bulkOids, v)
		}
		// sorting columnOids and bulkOids to make them deterministic for testing purpose
		sort.Strings(columnOids)
		sort.Strings(bulkOids)
		results, err := session.GetBulk(bulkOids)
		log.Debugf("fetchColumnOidsWithBatching() results: %v", results)
		if err != nil {
			return nil, fmt.Errorf("GetBulk failed: %s", err)
		}
		values, nextOids := resultToColumnValues(columnOids, results)
		for columnOid, columnValues := range values {
			for oid, value := range columnValues {
				if _, ok := returnValues[columnOid]; !ok {
					returnValues[columnOid] = make(map[string]snmpValue)
				}
				returnValues[columnOid][oid] = value
			}
		}
		curOids = nextOids
	}
	return returnValues, nil
}

func getOidsMapKeys(oidsMap map[string]string) []string {
	keys := make([]string, len(oidsMap))
	i := 0
	for k := range oidsMap {
		keys[i] = k
		i++
	}
	return keys
}
