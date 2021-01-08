package snmp

import (
	"fmt"
	"github.com/DataDog/datadog-agent/pkg/util/log"
	"sort"
)

func fetchValues(session sessionAPI, config snmpConfig) (*snmpValues, error) {
	scalarResults, err := fetchScalarOidsByBatch(session, config.OidConfig.scalarOids, config.OidBatchSize)
	if err != nil {
		return &snmpValues{}, fmt.Errorf("SNMPGET error: %v", err)
	}

	oids := make(map[string]string)
	for _, value := range config.OidConfig.columnOids {
		oids[value] = value
	}
	columnResults, err := fetchColumnOids(session, oids, config.OidBatchSize)
	if err != nil {
		return &snmpValues{}, fmt.Errorf("SNMPBULK error: %v", err)
	}

	return &snmpValues{scalarResults, columnResults}, nil
}

func fetchScalarOids(session sessionAPI, oids []string) (map[string]snmpValue, error) {
	// Get results
	log.Debugf("fetchScalarOidsByBatch() oids: %v", oids)
	results, err := session.Get(oids)
	log.Debugf("fetchColumnOids() results: %v", results)
	if err != nil {
		return nil, fmt.Errorf("error getting oids: %s", err.Error())
	}
	return resultToScalarValues(results), nil
}

func fetchScalarOidsByBatch(session sessionAPI, oids []string, oidBatchSize int) (map[string]snmpValue, error) {
	// Get results
	// TODO: Improve batching algorithm and make it more readable
	retValues := make(map[string]snmpValue)

	// Create batches of column oids
	batches, err := makeStringBatches(oids, oidBatchSize)
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

func fetchColumnOids(session sessionAPI, oids map[string]string, oidBatchSize int) (map[string]map[string]snmpValue, error) {
	// Get results
	// TODO: Improve batching algorithm and make it more readable

	retValues := make(map[string]map[string]snmpValue)

	// Create list of column oids from keys
	columnOids := make([]string, 0, len(oids))
	for k := range oids {
		columnOids = append(columnOids, k)
	}

	// Create batches of column oids
	batches, err := makeStringBatches(columnOids, oidBatchSize)
	if err != nil {
		return nil, fmt.Errorf("failed to create column oid batches: %s", err)
	}

	for _, batchColumnOids := range batches {
		oidsToFetch := make(map[string]string)
		for _, oid := range batchColumnOids {
			oidsToFetch[oid] = oids[oid]
		}

		results, err := fetchColumnOidsOneBatch(session, oidsToFetch)
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

// fetchColumnOidsOneBatch has an `oids` argument representing a `map[string]string`,
// the key of the map is the column oid, and the value is the oid used to fetch the next value for the column.
// The value oid might be equal to column oid or a row oid of the same column.
func fetchColumnOidsOneBatch(session sessionAPI, oids map[string]string) (map[string]map[string]snmpValue, error) {
	// Returns map[columnOID]map[index]interface(float64 or string)
	// GetBulk results
	// TODO:
	//   - make batches
	//   - GetBulk loop to get all rows
	returnValues := make(map[string]map[string]snmpValue)
	if len(oids) == 0 {
		return returnValues, nil
	}
	curOids := oids
	for {
		log.Debugf("fetchColumnOids() curOids  : %v", curOids)
		var columnOids, bulkOids []string
		for k, v := range curOids {
			columnOids = append(columnOids, k)
			bulkOids = append(bulkOids, v)
		}
		// sorting columnOids and bulkOids to make them deterministic for testing purpose
		sort.Strings(columnOids)
		sort.Strings(bulkOids)
		results, err := session.GetBulk(bulkOids)
		log.Debugf("fetchColumnOids() results: %v", results)
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
		if len(nextOids) == 0 {
			break
		}
		curOids = nextOids
	}
	return returnValues, nil
}
