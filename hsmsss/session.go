package hsmsss

import (
	"fmt"

	"github.com/arloliu/go-secs/hsms"
	"github.com/arloliu/go-secs/logger"
)

// Session represents an HSMS-SS session within an HSMS-SS connection.
// It provides methods for interacting with the session, sending and receiving messages,
// handling data messages, and managing connection state change handlers.
//
// It implements the hsms.Session interface.
type Session struct {
	hsms.BaseSession
	id       uint16
	hsmsConn *Connection
	cfg      *ConnectionConfig
	logger   logger.Logger

	dataMsgChans    []chan *hsms.DataMessage
	dataMsgHandlers []hsms.DataMessageHandler
}

// NewSession creates a new HSMS-SS Session with the specified session ID and associated Connection.
// It initializes the session's internal state and assigns HSMS-SS specific implementations to the base session methods.
func NewSession(id uint16, hsmsConn *Connection) *Session {
	session := &Session{
		id:              id,
		hsmsConn:        hsmsConn,
		cfg:             hsmsConn.cfg,
		logger:          hsmsConn.logger,
		dataMsgChans:    make([]chan *hsms.DataMessage, 0),
		dataMsgHandlers: make([]hsms.DataMessageHandler, 0),
	}

	// register HSMS-SS specific implementations to base session
	session.RegisterIDFunc(session.ID)
	session.RegisterSendMessageFunc(session.SendMessage)
	session.RegisterSendMessageAsyncFunc(session.SendMessageAsync)

	return session
}

// ID returns the session ID for this HSMS-SS session.
func (s *Session) ID() uint16 {
	return s.id
}

// SendMessage sends an HSMS message through the associated HSMS-SS connection and waits for its reply.
// It returns the received reply message and an error if any occurred during sending or receiving.
func (s *Session) SendMessage(msg hsms.HSMSMessage) (hsms.HSMSMessage, error) {
	return s.hsmsConn.sendMsg(msg)
}

// SendMessageAsync sends an HSMS message through the associated HSMS-SS connection asynchronously.
// It sends the message and its reply to the specified channel when received.
func (s *Session) SendMessageAsync(msg hsms.HSMSMessage) error {
	return s.hsmsConn.sendMsgAsync(msg)
}

// AddConnStateChangeHandler adds one or more ConnStateChangeHandler functions to be invoked when the connection state changes.
//
// Notes:
//   - The handler is responsible for processing the state change and taking appropriate action.
//   - The handler should not block the channel to prevent message loss.
//   - The handler should be registered before the session is opened.
//   - The handlers are invoked in the order they are added.
//   - The session will broadcast state information to all handlers' channels.
//
// Example:
//
//	session.AddConnStateChangeHandler(func(state hsms.ConnState) {
//		switch state {
//		case hsms.ConnStateConnected:
//			// handle connected state
//			case hsms.ConnStateDisconnected:
//			// handle disconnected state
//		}
//	})
func (s *Session) AddConnStateChangeHandler(handlers ...hsms.ConnStateChangeHandler) {
	s.hsmsConn.stateMgr.AddHandler(handlers...)
}

// AddDataMessageHandler adds one or more DataMessageHandler functions to be invoked when a data message is received.
//
// It creates a channel for each handler to receive messages, and it's used to handle data messages asynchronously.
//
// Notes:
//   - The handler is responsible for processing the message and sending a reply if necessary.
//   - The handler should not block the channel to prevent message loss.
//   - The handler should be registered before the session is opened.
//   - The handlers are invoked in the order they are added.
//   - The session will broadcast messages to all handlers' channels.
//
// Example:
//
//	session.AddDataMessageHandler(func(msg *hsms.DataMessage, session hsms.Session) {
//	    if msg.FunctionCode()%2 == 1 {
//	        // handle request message
//	        err := session.ReplyDataMessage(msg, msg.Item())
//	        if err != nil {
//	            // handle reply error
//	        }
//	        return
//	    }
//
//	    // handle response message
//	})
func (s *Session) AddDataMessageHandler(handlers ...hsms.DataMessageHandler) {
	for _, handler := range handlers {
		s.dataMsgChans = append(s.dataMsgChans, make(chan *hsms.DataMessage, s.cfg.dataMsgQueueSize))
		s.dataMsgHandlers = append(s.dataMsgHandlers, handler)
	}
}

func (s *Session) startDataMsgTasks() {
	for i, handler := range s.dataMsgHandlers {
		name := fmt.Sprintf("dataMsgTask-%d", i+1)
		s.hsmsConn.taskMgr.StartRecvDataMsg(name, handler, s, s.dataMsgChans[i])
	}
}

// recvDataMsg broadcast message to all data message handlers' channel
func (s *Session) recvDataMsg(msg *hsms.DataMessage) {
	for _, dataMsgChan := range s.dataMsgChans {
		dataMsgChan <- msg
	}
}

func (s *Session) separateSession() {
	// send separate.req message, in HSMS-SS, the session ID is always 0xffff
	msg := hsms.NewSeparateReq(0xffff, hsms.GenerateMsgSystemBytes())
	s.logger.Debug("send separate.req message and wait it to be sent", "method", "separateSession", "id", msg.ID())
	err := s.hsmsConn.sendMsgSync(msg)
	if err != nil && !isNetError(err) {
		s.logger.Warn("failed to send separate control message", "method", "separateSession", "id", msg.ID(), "error", err)
	}
}

func (s *Session) selectSession() error {
	s.logger.Debug("send select.req", "method", "selectSession")
	// select request, in HSMS-SS, the session ID is always 0xffff
	msg := hsms.NewSelectReq(0xffff, hsms.GenerateMsgSystemBytes())
	replyMsg, err := s.hsmsConn.sendControlMsg(msg, true)
	if err != nil {
		return err
	}

	if replyMsg == nil || replyMsg.Type() != hsms.SelectRspType {
		return hsms.ErrInvalidRspMsg
	}

	// read select status
	selectStatus := replyMsg.Header()[3]
	switch selectStatus {
	case 0:
		// success
		s.logger.Debug("connection selected", "session_id", replyMsg.SessionID(), "type", replyMsg.Type())
		return nil
	default:
		s.logger.Warn("failed to select session", "session_id", replyMsg.SessionID(), "select_status", selectStatus)
		return hsms.ErrSelectFailed
	}
}
