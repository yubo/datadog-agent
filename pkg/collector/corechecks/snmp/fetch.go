package snmp

import (
	"github.com/DataDog/datadog-agent/pkg/util/log"
	"sort"
)

func fetchScalarOids(session sessionAPI, oids []string) (map[string]snmpValue, error) {
	// Get results
	// TODO: make batches
	log.Debugf("fetchScalarOids() oids: %v", oids)
	results, err := session.Get(oids)
	log.Debugf("fetchColumnOids() results: %v", results)
	if err != nil {
		return nil, err
	}
	return resultToScalarValues(results), nil
}

func fetchColumnOids(session sessionAPI, oids map[string]string) (map[string]map[string]snmpValue, error) {
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
			return nil, err
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
