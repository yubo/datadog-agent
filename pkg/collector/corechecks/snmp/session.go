package snmp

import (
	"github.com/soniah/gosnmp"
	"time"
)

type sessionAPI interface {
	Configure(config snmpConfig)
	Connect() error
	Close() error
	Get(oids []string) (result *gosnmp.SnmpPacket, err error)
	GetBulk(oids []string) (result *gosnmp.SnmpPacket, err error)
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
		Timeout:   time.Duration(config.Timeout) * time.Second,
		Retries:   config.Retries,
	}
}

func (s *snmpSession) Connect() error {
	return s.gosnmpInst.Connect()
}

func (s *snmpSession) Close() error {
	return s.gosnmpInst.Conn.Close()
}

func (s *snmpSession) Get(oids []string) (result *gosnmp.SnmpPacket, err error) {
	return s.gosnmpInst.Get(oids)
}

func (s *snmpSession) GetBulk(oids []string) (result *gosnmp.SnmpPacket, err error) {
	if len(oids) == 0 { // TODO: test me
		return &gosnmp.SnmpPacket{}, nil
	}
	return s.gosnmpInst.GetBulk(oids, 0, 10)
}
