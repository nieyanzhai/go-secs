package hsms

import (
	"github.com/arloliu/go-secs/secs2"
)

// Type constants representing the different types of HSMS messages.
// These constants categorize messages based on their primary function and role in the HSMS protocol.
const (
	UndefiniedMsgType = -1 // Undefeind stream session type
	DataMsgType       = 0  // Data message containing SECS-II data
	SelectReqType     = 1  // Select request control message
	SelectRspType     = 2  // Select response control message
	DeselectReqType   = 3  // Deselect request control message
	DeselectRspType   = 4  // Deselect response control message
	LinkTestReqType   = 5  // Linktest request control message
	LinkTestRspType   = 6  // Linktest response control message
	RejectReqType     = 7  // Reject request control message
	SeparateReqType   = 9  // Separate request control message
)

var hsmsMsgTypeMap = map[int]string{
	DataMsgType:       "data.msg",
	SelectReqType:     "select.req",
	SelectRspType:     "select.rsp",
	DeselectReqType:   "deselect.req",
	DeselectRspType:   "deselect.rsp",
	LinkTestReqType:   "linktest.req",
	LinkTestRspType:   "linktest.rsp",
	RejectReqType:     "reject.req",
	SeparateReqType:   "separate.req",
	UndefiniedMsgType: "undefined",
}

// HSMSMessage represents a message in the HSMS (High-Speed SECS Message Services) protocol.
// It extends the SECS2Message interface to include HSMS-specific attributes and functionalities.
//
// HSMS messages are categorized into:
//   - Data Message: Used for exchanging SECS-II data between the host and equipment.
//   - Control Message: Used for managing the HSMS connection itself (e.g., session control, link testing).
//
// This interface provides methods for accessing common HSMS message attributes and converting the message
// into its specific data or control message representation.
type HSMSMessage interface {
	secs2.SECS2Message

	// Type returns the HSMS message type, which can be one of the following constants:
	//  - hsms.DataMsgType
	//  - hsms.SelectReqType
	//  - hsms.SelectRspType
	//  - hsms.DeselectReqType
	//  - hsms.DeselectRspType
	//  - hsms.LinkTestReqType
	//  - hsms.LinkTestRspType
	//  - hsms.RejectReqType
	//  - hsms.SeparateReqType
	Type() int

	// SessionID returns the session ID for the HSMS message.
	SessionID() uint16

	// ID returns a numeric representation of the system bytes (message ID).
	ID() uint32

	// SystemBytes returns the 4-byte system bytes (message ID).
	SystemBytes() []byte

	// Header returns the 10-byte HSMS message header.
	Header() []byte

	// ToBytes serializes the HSMS message into its byte representation for transmission.
	ToBytes() []byte

	// IsControlMessage returns if the message is control message.
	IsControlMessage() bool
	// ToControlMessage converts the message to an HSMS control message if applicable.
	// It returns a pointer to the ControlMessage and a boolean indicating if the conversion was successful.
	ToControlMessage() (*ControlMessage, bool)

	// IsDataMessage returns if the message is data message.
	IsDataMessage() bool
	// ToDataMessage converts the message to an HSMS data message if applicable.
	// It returns a pointer to the DataMessage and a boolean indicating if the conversion was successful.
	ToDataMessage() (*DataMessage, bool)

	// Free releases the message and its associated resources back to the pool.
	// After calling Free, the message should not be accessed again.
	Free()

	// Clone creates a deep copy of the message, allowing modifications to the clone without affecting the original message.
	Clone() HSMSMessage
}

var sfQuote string = "'"

func UseStreamFunctionNoQuote() {
	sfQuote = ""
}

func UseStreamFunctionSingleQuote() {
	sfQuote = "'"
}

func UseStreamFunctionDoubleQuote() {
	sfQuote = "\""
}

func StreamFunctionQuote() string {
	return sfQuote
}

// MsgInfo returns a structued message information without SML string.
func MsgInfo(msg HSMSMessage, keyValues ...any) []any {
	return msgInfo(msg, false, keyValues...)
}

// MsgInfoSML returns a structued message information with SML string.
func MsgInfoSML(msg HSMSMessage, keyValues ...any) []any {
	return msgInfo(msg, true, keyValues...)
}

func msgInfo(msg HSMSMessage, sml bool, keyValues ...any) []any {
	msgType, ok := hsmsMsgTypeMap[msg.Type()]
	if !ok {
		msgType = "undefined"
	}

	info := []any{
		"id", msg.ID(),
		"type", msgType,
		"s", msg.StreamCode(),
		"f", msg.FunctionCode(),
	}

	if sml && msg.Item() != nil {
		info = append(info, "sml", msg.Item().ToSML())
	}

	result := make([]any, 0, len(keyValues)+len(info))
	result = append(result, keyValues...)
	result = append(result, info...)

	return result
}
