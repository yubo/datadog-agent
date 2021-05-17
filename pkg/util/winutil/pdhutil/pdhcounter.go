// Unless explicitly stated otherwise all files in this repository are licensed
// under the Apache License Version 2.0.
// This product includes software developed at Datadog (https://www.datadoghq.com/).
// Copyright 2016-present Datadog, Inc.
// +build windows

package pdhutil

import (
	"fmt"

	"github.com/DataDog/datadog-agent/pkg/util/log"
)

// For testing
var (
	pfnMakeCounterSetInstances          = makeCounterSetIndexes
	pfnPdhOpenQuery                     = PdhOpenQuery
	pfnPdhAddEnglishCounter             = PdhAddEnglishCounter
	pfnPdhCollectQueryData              = PdhCollectQueryData
	pfnPdhEnumObjectItems               = pdhEnumObjectItems
	pfnPdhRemoveCounter                 = PdhRemoveCounter
	pfnPdhLookupPerfNameByIndex         = pdhLookupPerfNameByIndex
	pfnPdhGetFormattedCounterValueFloat = pdhGetFormattedCounterValueFloat
	pfnPdhCloseQuery                    = PdhCloseQuery
	pfnPdhMakeCounterPath               = pdhMakeCounterPath
)

// CounterInstanceVerify is a callback function called by GetCounterSet for each
// instance of the counter.  Implementation should return true if that instance
// should be included, false otherwise
type CounterInstanceVerify func(string) bool

// PdhCounterSet is the object which represents a pdh counter set.
type PdhCounterSet struct {
	className string
	query     PDH_HQUERY

	counterName string
}

// PdhSingleInstanceCounterSet is a specialization for single instance counters
type PdhSingleInstanceCounterSet struct {
	PdhCounterSet
	singleCounter PDH_HCOUNTER
}

// PdhMultiInstanceCounterSet is a specialization for a multiple instance counter
type PdhMultiInstanceCounterSet struct {
	PdhCounterSet
	requestedCounterName string
	requestedInstances   map[string]bool
	countermap           map[string]PDH_HCOUNTER // map instance name to counter handle
	verifyfn             CounterInstanceVerify
}

// Initialize initializes a counter set object
func (p *PdhCounterSet) Initialize(className string) error {
	p.className = className
	winerror := pfnPdhOpenQuery(uintptr(0), uintptr(0), &p.query)
	if ERROR_SUCCESS != winerror {
		return fmt.Errorf("Failed to open PDH query handle %d", winerror)
	}
	return nil
}

// GetSingleInstanceCounter returns a single instance counter object for the given counter class
func GetSingleInstanceCounter(className, counterName string) (*PdhSingleInstanceCounterSet, error) {
	var p PdhSingleInstanceCounterSet
	if err := p.Initialize(className); err != nil {
		return nil, err
	}
	path, err := pfnPdhMakeCounterPath("", className, "", counterName)
	if err != nil {
		log.Warnf("Failed pdhEnumObjectItems %v", err)
		return nil, err
	}
	winerror := pfnPdhAddEnglishCounter(p.query, path, uintptr(0), &p.singleCounter)
	if ERROR_SUCCESS != winerror {
		return nil, fmt.Errorf("Failed to add single counter %d", winerror)
	}

	// do the initial collect now
	pfnPdhCollectQueryData(p.query)
	return &p, nil
}

// GetMultiInstanceCounter returns a multi-instance counter object for the given counter class
func GetMultiInstanceCounter(className, counterName string, requestedInstances *[]string, verifyfn CounterInstanceVerify) (*PdhMultiInstanceCounterSet, error) {
	var p PdhMultiInstanceCounterSet
	if err := p.Initialize(className); err != nil {
		return nil, err
	}
	p.countermap = make(map[string]PDH_HCOUNTER)
	p.verifyfn = verifyfn
	p.requestedCounterName = counterName

	// save the requested instances
	if requestedInstances != nil && len(*requestedInstances) > 0 {
		p.requestedInstances = make(map[string]bool)
		for _, inst := range *requestedInstances {
			p.requestedInstances[inst] = true
		}
	}
	if err := p.MakeInstanceList(); err != nil {
		return nil, err
	}
	return &p, nil

}

// MakeInstanceList walks the list of available instances, and adds new
// instances that have appeared since the last check run
func (p *PdhMultiInstanceCounterSet) MakeInstanceList() error {
	added := false
	for inst := range p.requestedInstances {
		if p.verifyfn != nil {
			if p.verifyfn(inst) == false {
				// not interested, moving on
				continue
			}
		}
		path, err := pfnPdhMakeCounterPath("", p.className, inst, p.requestedCounterName)
		if err != nil {
			log.Debugf("Failed tomake counter path %s %s", p.counterName, inst)
			continue
		}
		var hc PDH_HCOUNTER
		winerror := pfnPdhAddEnglishCounter(p.query, path, uintptr(0), &hc)
		if ERROR_SUCCESS != winerror {
			log.Debugf("Failed to add counter path %s", path)
			continue
		}
		log.Debugf("Adding missing counter instance %s", inst)
		p.countermap[inst] = hc
		added = true
	}
	if added {
		// do the initial collect now
		pfnPdhCollectQueryData(p.query)
	}
	return nil
}

//RemoveInvalidInstance removes an instance from the counter that is no longer valid
func (p *PdhMultiInstanceCounterSet) RemoveInvalidInstance(badInstance string) {
	hc := p.countermap[badInstance]
	if hc != PDH_HCOUNTER(0) {
		log.Debugf("Removing non-existent counter instance %s", badInstance)
		pfnPdhRemoveCounter(hc)
		delete(p.countermap, badInstance)
	} else {
		log.Debugf("Instance handle not found")
	}
}

// GetAllValues returns the data associated with each instance in a query.
func (p *PdhMultiInstanceCounterSet) GetAllValues() (values map[string]float64, err error) {
	values = make(map[string]float64)
	err = nil
	var removeList []string
	pfnPdhCollectQueryData(p.query)
	for inst, hcounter := range p.countermap {
		var retval float64
		retval, err = pfnPdhGetFormattedCounterValueFloat(hcounter)
		if err != nil {
			switch err.(type) {
			case *ErrPdhInvalidInstance:
				removeList = append(removeList, inst)
				log.Debugf("Got invalid instance for %s %s", p.requestedCounterName, inst)
				err = nil
				continue
			default:
				log.Debugf("Other Error getting all values %s %s %v", p.requestedCounterName, inst, err)
				return
			}
		}
		values[inst] = retval
	}
	for _, inst := range removeList {
		p.RemoveInvalidInstance(inst)
	}
	// check for newly found instances
	p.MakeInstanceList()
	return
}

// GetValue returns the data associated with a single-value counter
func (p *PdhSingleInstanceCounterSet) GetValue() (val float64, err error) {
	if p.singleCounter == PDH_HCOUNTER(0) {
		return 0, fmt.Errorf("Not a single-value counter")
	}
	pfnPdhCollectQueryData(p.query)
	return pfnPdhGetFormattedCounterValueFloat(p.singleCounter)

}

// Close closes the query handle, freeing the underlying windows resources.
func (p *PdhCounterSet) Close() {
	PdhCloseQuery(p.query)
}
