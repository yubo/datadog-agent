package snmp

import "github.com/DataDog/datadog-agent/pkg/util/log"

func fetchScalarOids(session sessionAPI, oids []string) (map[string]interface{}, error) {
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

func fetchColumnOids(session sessionAPI, oids []string) (map[string]map[string]interface{}, error) {
	// Returns map[columnOID]map[index]interface(float64 or string)
	// GetBulk results
	// TODO:
	//   - make batches
	//   - GetBulk loop to get all rows
	log.Debugf("fetchColumnOids() oids  : %v", oids)
	resuts, err := session.GetBulk(oids)
	log.Debugf("fetchColumnOids() resuts: %v", resuts)
	if err != nil {
		return nil, err
	}
	return resultToColumnValues(oids, resuts), nil
}
