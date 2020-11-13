package snmp

import (
	"github.com/soniah/gosnmp"
	"time"
)

type sessionAPI interface {
	Configure(config snmpConfig)
	Get(oids []string) (result *gosnmp.SnmpPacket, err error)
	Connect() error
	Close() error
}

type snmpSession struct {
	gosnmpInst gosnmp.GoSNMP
}

func (s *snmpSession) Configure(config snmpConfig) {
	s.gosnmpInst = gosnmp.GoSNMP{
		Target:    config.IPAddress,
		Port:      config.Port,
		Community: config.CommunityString,
		Version:   config.SnmpVersion,
		// TODO: implement following configs
		Timeout:            time.Duration(2) * time.Second,
		Retries:            3,
		ExponentialTimeout: true,
		MaxOids:            100,
	}
}

func (s *snmpSession) Get(oids []string) (result *gosnmp.SnmpPacket, err error) {
	return s.gosnmpInst.Get(oids)
}

func (s *snmpSession) Connect() error {
	return s.gosnmpInst.Connect()
}

func (s *snmpSession) Close() error {
	return s.gosnmpInst.Conn.Close()
}
