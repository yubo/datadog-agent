package snmp

import (
	"fmt"
	"github.com/gosnmp/gosnmp"
	"time"
)

type sessionAPI interface {
	Configure(config snmpConfig) error
	Connect() error
	Close() error
	Get(oids []string) (result *gosnmp.SnmpPacket, err error)
	GetBulk(oids []string) (result *gosnmp.SnmpPacket, err error)
}

type snmpSession struct {
	gosnmpInst gosnmp.GoSNMP
}

func (s *snmpSession) Configure(config snmpConfig) error {
	maxOids := gosnmp.MaxOids
	if config.oidBatchSize > gosnmp.MaxOids {
		return fmt.Errorf("config oidBatchSize (%d) cannot higher than gosnmp.MaxOids: %d", config.oidBatchSize, maxOids)
	}
	snmpVersion, err := parseVersion(config.snmpVersion)
	if err != nil {
		return err
	}
	gosnmpInst := gosnmp.GoSNMP{
		Target:  config.ipAddress,
		Port:    config.port,
		Version: snmpVersion,
		Timeout: time.Duration(config.timeout) * time.Second,
		Retries: config.retries,
		MaxOids: maxOids,
		// Uncomment following line for debugging
		//Logger:  defaultLog.New(os.Stdout, "", 0),
	}
	switch snmpVersion {
	case gosnmp.Version2c, gosnmp.Version1:
		gosnmpInst.Community = config.communityString
	case gosnmp.Version3:
		authProtocol, err := getAuthProtocol(config.authProtocol)
		if err != nil {
			return err
		}

		privProtocol, err := getPrivProtocol(config.privProtocol)
		if err != nil {
			return err
		}

		msgFlags := gosnmp.NoAuthNoPriv
		if privProtocol != gosnmp.NoPriv {
			msgFlags = gosnmp.AuthPriv
		} else if authProtocol != gosnmp.NoAuth {
			msgFlags = gosnmp.AuthNoPriv
		}

		gosnmpInst.MsgFlags = msgFlags
		gosnmpInst.ContextName = config.contextName
		gosnmpInst.SecurityModel = gosnmp.UserSecurityModel
		gosnmpInst.SecurityParameters = &gosnmp.UsmSecurityParameters{
			UserName:                 config.user,
			AuthenticationProtocol:   authProtocol,
			AuthenticationPassphrase: config.authKey,
			PrivacyProtocol:          privProtocol,
			PrivacyPassphrase:        config.privKey,
		}
	}
	s.gosnmpInst = gosnmpInst
	return nil
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
	if len(oids) == 0 {
		return &gosnmp.SnmpPacket{}, nil
	}
	return s.gosnmpInst.GetBulk(oids, 0, 10)
}

func fetchSysObjectID(session sessionAPI) (string, error) {
	result, err := session.Get([]string{"1.3.6.1.2.1.1.2.0"})
	if err != nil {
		return "", fmt.Errorf("cannot get sysobjectid: %s", err)
	}
	_, value, err := getValueFromPDU(result.Variables[0])
	if err != nil {
		return "", fmt.Errorf("error getting value from pdu: %s", err)
	}
	return value.toString(), err
}
