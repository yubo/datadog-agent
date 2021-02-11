package testutil

import (
	"testing"

	"github.com/cihub/seelog"
)

type TestLogger struct {
	tt *testing.T
}

func NewTestLogger(t *testing.T) (seelog.LoggerInterface, error) {
	log, err := seelog.LoggerFromCustomReceiver(&TestLogger{t})
	t.Cleanup(log.Close)
	return log, err
}

func (t *TestLogger) ReceiveMessage(message string, level seelog.LogLevel, context seelog.LogContextInterface) error {
	t.tt.Logf("[%s] %s", level.String(), message)
	return nil
}

func (t *TestLogger) AfterParse(initArgs seelog.CustomReceiverInitArgs) error {
	return nil
}

func (t *TestLogger) Close() error {
	t.tt = nil
	return nil
}

func (t *TestLogger) Flush() {
}
