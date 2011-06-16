// Copyright 2010 GoDCCP Authors. All rights reserved.
// Use of this source code is governed by a 
// license that can be found in the LICENSE file.

package dccp

import (
	"log"
	"os"
)

func (c *Conn) writeCCID(h *Header) *Header {
	// HC-Sender CCID
	ccval, sropts := c.scc.OnWrite(h.Type, h.X, h.SeqNo)
	if !validateCCIDSenderToReceiver(sropts) {
		panic("sender congestion control writes disallowed options")
	}
	h.CCVal = ccval
	// HC-Receiver CCID
	rsopts := c.rcc.OnWrite(h.Type, h.X, h.SeqNo)
	if !validateCCIDReceiverToSender(rsopts) {
		panic("receiver congestion control writes disallowed options")
	}
	// XXX: Also check option compatibility with respect to packet type (Data vs. other)
	h.Options = append(h.Options, append(sropts, rsopts...)...)
	return h
}

// inject adds the packet h to the outgoing non-Data pipeline, without blocking.  The
// pipeline is flushed continuously respecting the CongestionControl's rate-limiting policy.
//
// inject is called at most once (currently) from inside readLoop and inside a lock
// on Conn, so it must not block, hence writeNonData has buffer space
func (c *Conn) inject(h *Header) {
	c.writeNonDataLk.Lock()
	defer c.writeNonDataLk.Unlock()

	if c.writeNonData == nil {
		return
	}
	// Dropping a nil is OK, since it happens only if there are other packets in the queue
	if len(c.writeNonData) < cap(c.writeNonData) {
		if h != nil {
			h = c.writeCCID(h)
		}
		c.writeNonData <- h
		if h != nil {
			c.logWriteHeaderLocked(h)
		}
	} else {
		log.Printf("dropping non-data, congestion rate too slow\n")
	}
}

func (c *Conn) write(h *Header) os.Error {
	c.scc.Strobe()
	return c.hc.WriteHeader(h)
}

// writeLoop() sends headers incoming on the writeData and writeNonData channels, while
// giving priority to writeNonData. It continues to do so until writeNonData is closed.
func (c *Conn) writeLoop(writeNonData chan *Header, writeData chan []byte) {

	// The presence of multiple loops below allows user calls to WriteBlock to
	// block in "writeNonData <-" while the connection moves into a state where
	// it accepts app data (in Loop_II)

	// This loop is active until state OPEN or PARTOPEN is observed, when a
	// transition to Loop II_is made
	Loop_I:

	for {
		h, ok := <-writeNonData
		if !ok {
			// Closing writeNonData means that the Conn is done and dead
			goto Exit
		}
		// We'll allow nil headers, since they can be used to trigger unblock
		// from the above send operator and (without resulting into an actual
		// send) activate the state check after the "if" statement below
		if h != nil {
			err := c.write(h)
			// If the underlying layer is broken, abort
			if err != nil {
				c.abortQuietly()
				goto Exit
			}
		}
		c.Lock()
		state := c.socket.GetState()
		c.Unlock()
		switch state {
		case OPEN, PARTOPEN:
			goto Loop_II
		}
		continue Loop_I
	}

	// This loop is active until writeData is not closed
	Loop_II:

	for {
		var h *Header
		var ok bool
		var appData []byte
		select {
		// Note that non-Data packets take precedence
		case h, ok = <-writeNonData:
			if !ok {
				// Closing writeNonData means that the Conn is done and dead
				goto Exit
			}
		case appData, ok = <-writeData:
			if !ok {
				// When writeData is closed, we transition to the 3rd loop,
				// which accepts only non-Data packets
				goto Loop_III
			}
			// By virtue of being in Loop_II (which implies we have been or are in OPEN
			// or PARTOPEN), we know that some packets of the other side have been
			// received, and so AckNo can be filled in meaningfully (below) in the
			// DataAck packet

			// We allow 0-length app data packets. No reason not to.
			// XXX: I am not sure if Header.Data == nil (rather than
			// Header.Data = []byte{}) would cause a problem in Header.Write
			// It should be that it doesn't. Must verify this.
			c.Lock()
			h = c.generateDataAck(appData)
			h = c.writeCCID(h)
			c.Unlock()
		}
		if h != nil {
			err := c.write(h)
			if err != nil {
				c.abortQuietly()
				goto Exit
			}
		}
	}

	// This loop is active until writeNonData is not closed
	Loop_III:

	for {
		h, ok := <-writeNonData
		if !ok {
			// Closing writeNonData means that the Conn is done and dead
			goto Exit
		}
		// We'll allow nil headers, since they can be used to trigger unblock
		// from the above send operator
		if h != nil {
			err := c.write(h)
			// If the underlying layer is broken, abort
			if err != nil {
				c.abortQuietly()
				goto Exit
			}
		}
	}

	Exit:
}