// Copyright 2010 GoDCCP Authors. All rights reserved.
// Use of this source code is governed by a 
// license that can be found in the LICENSE file.

package ccid3

import (
	"os"
	"github.com/petar/GoDCCP/dccp"
)

type sender struct {
	dccp.Mutex
	phase int
	rtt   int64 // RTT estimate if available, negative otherwise

	windowCounter
	strober
}

// Phases of the congestion control mechanism
const (
	INIT = iota
	SLOWSTART
	EQUATION
	CLOSED
)

// NOTES
//
// The sender starts in a slow-start phase, roughly doubling its allowed sending rate each
// round-trip time.  The slow-start phase is ended by the receiver's report of a data packet drop or
// mark, after which the sender uses the loss event rate to calculate its allowed sending rate.

// GetID() returns the CCID of this congestion control algorithm
func (s *sender) GetID() byte { return dccp.CCID3 }

// GetCCMPS returns the Congestion Control Maximum Packet Size, CCMPS. Generally, PMTU <= CCMPS
func (s *sender) GetCCMPS() int32 {
	?
}

// GetRTT returns the Round-Trip Time as measured by this CCID
func (s *sender) GetRTT() int64 {
	?
}

// Open tells the Congestion Control that the connection has entered
// OPEN or PARTOPEN state and that the CC can now kick in. Before the
// call to Open and after the call to Close, the Strobe function is
// expected to return immediately.
func (s *sender) Open() {
	?
}

// Conn calls OnWrite before a packet is sent to give CongestionControl
// an opportunity to add CCVal and options to an outgoing packet
// NOTE: If the CC is not active, OnWrite should return 0, nil.
func (s *sender) OnWrite(htype byte, x bool, seqno int64) (ccval byte, options []*dccp.Option) {
	?
}

// Conn calls OnRead after a packet has been accepted and validated
// If OnRead returns ErrDrop, the packet will be dropped and no further processing
// will occur. If OnRead returns ResetError, the connection will be reset.
// NOTE: If the CC is not active, OnRead MUST return nil.
func (s *sender) OnRead(htype byte, x bool, seqno int64, options []*dccp.Option) os.Error {
	?
}

// Strobe blocks until a new packet can be sent without violating the
// congestion control rate limit. 
// NOTE: If the CC is not active, Strobe MUST return immediately.
func (s *sender) Strobe() {
	?
}

// OnIdle is called periodically, giving the CC a chance to:
// (a) Request a connection reset by returning a CongestionReset, or
// (b) Request the injection of an Ack packet by returning a CongestionAck
// NOTE: If the CC is not active, OnIdle MUST to return nil.
func (s *sender) OnIdle() os.Error {
	?
}

// Close terminates the half-connection congestion control when it is not needed any longer
func (s *sender) Close() {
	?
}
