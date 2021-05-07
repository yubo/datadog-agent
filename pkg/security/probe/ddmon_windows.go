// +build windows

package probe

// #include <windows.h>
// #include <wchar.h>
// #include <WinNT.h>
import "C"

import (
	"github.com/DataDog/datadog-agent/pkg/util/log"
	"golang.org/x/sys/windows"
	"syscall"
	"unicode/utf16"
	"unsafe"
	"errors"
)

var (
	ddmonAPI                 = windows.NewLazyDLL("ddmondll.dll")
	procOpenDdmonDriver      = ddmonAPI.NewProc("OpenDdmonDriver")
	procCloseDdmonDriver     = ddmonAPI.NewProc("CloseDdmonDriver")
	procStartMonitoring      = ddmonAPI.NewProc("StartMonitoring")
	procStopMonitoring       = ddmonAPI.NewProc("StopMonitoring")
	procHackTestPolicySimple = ddmonAPI.NewProc("HackTestPolicySimple")
)

const maxRunes = 1<<30 - 1

// Common data shared between internal and public data structures.
type DdmonFileActivityCoreRecord struct {
	OriginatingTime uint64
	CompletionTime  uint64

	DeviceObject uint64
	FileObject   uint64
	Transaction  uint64

	ProcessId uint64
	ThreadId  uint64

	IrpFlags uint32
	Flags    uint32

	CallbackMajorId uint8
	CallbackMinorId uint8

	EcpCount     uint32
	KnownEcpMask uint32

	Status int32

	Information uint64

	Arg1 uint64
	Arg2 uint64
	Arg3 uint64
	Arg4 uint64
	Arg5 uint64
	Arg6 uint64
}

// Internal data structure, identical to the C code.
type ddmonInternalFileActivityRecord struct {
	CoreRecordInfo DdmonFileActivityCoreRecord
	UserName       C.PWSTR
	ProcessName    C.PWSTR
	FileName       C.PWSTR
}

// Public data structure.
type DdmonFileActivityRecord struct {
	CoreRecordInfo DdmonFileActivityCoreRecord
	UserName       string
	ProcessName    string
	FileName       string
}

type ddmonInternalFileActivityRecordPtr *ddmonInternalFileActivityRecord

// Public callback function definition.
type DdmonIoCallbackFunction func(records []DdmonFileActivityRecord) int32

// Ddmon class definition.
type Ddmon struct {
	extCallbackFunc          DdmonIoCallbackFunction
}

func isHresultFailed(ret uintptr) bool {
	i := int32(uint32(ret))
	return i < 0
}

func stringFromPCWSTR(pwstr C.PWSTR) string {
	if pwstr == nil {
		return ""
	}
	ptr := unsafe.Pointer(pwstr)
	sz := C.wcslen((*C.wchar_t)(ptr))
	wstr := (*[maxRunes]uint16)(ptr)[:sz:sz]
	return string(utf16.Decode(wstr))
}

// Internal callback function that converts internal data structure to public and invokes the registered callback.
func (m *Ddmon) ioCallback(records uintptr, numOfRecords uint32) uintptr {
	var result int32
	if numOfRecords == 0 {
		result = m.extCallbackFunc(nil)
	} else {
		extRecords := make([]DdmonFileActivityRecord, numOfRecords)
		for i := uint32(0); i < numOfRecords; i++ {
			r := *(*((*ddmonInternalFileActivityRecordPtr)(unsafe.Pointer(records + unsafe.Sizeof(records)*uintptr(i)))))
			extRecords[i].CoreRecordInfo = r.CoreRecordInfo
			extRecords[i].FileName = stringFromPCWSTR(r.FileName)
			extRecords[i].ProcessName = stringFromPCWSTR(r.ProcessName)
			extRecords[i].UserName = stringFromPCWSTR(r.UserName)
		}
		result = m.extCallbackFunc(extRecords)
	}
	return uintptr(uint32(result))
}

func NewDdmon(callbackFunc DdmonIoCallbackFunction) (*Ddmon, error) {
	if callbackFunc == nil {
		return nil, errors.New("null callback function")
	}

	m := &Ddmon{
		callbackFunc,
	}
	return m, nil
}

func (m *Ddmon) DdmonOpenDriver() error {
	ret, _, apiError := procOpenDdmonDriver.Call()
	if isHresultFailed(ret) {
		log.Errorf("could not open ddmon driver, LastError: %v\n", apiError)
		return apiError
	}
	return nil
}

func (m *Ddmon) DdmonCloseDriver() error {
	ret, _, apiError := procCloseDdmonDriver.Call()
	if isHresultFailed(ret) {
		log.Errorf("could not close ddmon driver, LastError: %v\n", apiError)
		return apiError
	}
	return nil
}

func (m *Ddmon) DdmonStartMonitoring() error {
	callback := syscall.NewCallback(m.ioCallback)
	ret, _, apiError := procStartMonitoring.Call(callback)
	if isHresultFailed(ret) {
		log.Errorf("could not start monitoring, LastError: %v\n", apiError)
		return apiError
	}
	return nil
}

func (m *Ddmon) DdmonStopMonitoring() error {
	ret, _, apiError := procStopMonitoring.Call()
	if isHresultFailed(ret) {
		log.Errorf("could not stop monitoring, LastError: %v\n", apiError)
		return apiError
	}
	return nil
}

func (m *Ddmon) DdmonSetTestRule() error {
	ret, _, apiError := procHackTestPolicySimple.Call()
	if isHresultFailed(ret) {
		log.Errorf("could not set test rule, LastError: %v\n", apiError)
		return apiError
	}
	return nil
}
