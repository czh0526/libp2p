package test_mocknet

import (
	"bytes"
	"errors"
	"io"
	"net"
	"sync/atomic"
	"time"

	inet "github.com/libp2p/go-libp2p-net"
	protocol "github.com/libp2p/go-libp2p-protocol"
)

type stream struct {
	write     *io.PipeWriter
	read      *io.PipeReader
	conn      *conn
	toDeliver chan *transportObject

	reset  chan struct{}
	close  chan struct{}
	closed chan struct{}

	writeErr error
	protocol atomic.Value
	stat     inet.Stat
}

var ErrReset error = errors.New("stream reset")
var ErrClosed error = errors.New("stream closed")

type transportObject struct {
	msg         []byte
	arrivalTime time.Time
}

func NewStream(w *io.PipeWriter, r *io.PipeReader, dir inet.Direction) *stream {
	s := &stream{
		read:      r,
		write:     w,
		reset:     make(chan struct{}, 1),
		close:     make(chan struct{}, 1),
		closed:    make(chan struct{}),
		toDeliver: make(chan *transportObject),
		stat:      inet.Stat{Direction: dir},
	}
	go s.transport()
	return s
}

func (s *stream) Read(b []byte) (int, error) {
	return s.read.Read(b)
}

func (s *stream) Write(p []byte) (n int, err error) {
	l := s.conn.link
	delay := l.GetLatency() + l.RateLimit(len(p))
	t := time.Now().Add(delay)

	cpy := make([]byte, len(p))
	copy(cpy, p)

	select {
	case <-s.closed:
		return 0, s.writeErr
	case s.toDeliver <- &transportObject{msg: cpy, arrivalTime: t}:
	}

	return len(p), nil
}

func (s *stream) Close() error {
	select {
	case s.close <- struct{}{}:
	default:
	}
	<-s.closed
	if s.writeErr != ErrClosed {
		return s.writeErr
	}
	return nil
}

func (s *stream) Reset() error {
	// 向远程发送重置信号
	s.write.CloseWithError(ErrReset)
	s.read.CloseWithError(ErrReset)

	// 向本地发送重置信号
	select {
	case s.reset <- struct{}{}:
	default:
	}
	// 等到关闭
	<-s.closed
	return nil
}

func (s *stream) Protocol() protocol.ID {
	p, _ := s.protocol.Load().(protocol.ID)
	return p
}

func (s *stream) SetProtocol(proto protocol.ID) {
	s.protocol.Store(proto)
}

func (s *stream) Stat() inet.Stat {
	return s.stat
}

func (s *stream) Conn() inet.Conn {
	return s.conn
}

func (s *stream) SetDeadline(t time.Time) error {
	return &net.OpError{
		Op:     "set",
		Net:    "pipe",
		Source: nil,
		Addr:   nil,
		Err:    errors.New("deadline not supported"),
	}
}

func (s *stream) SetReadDeadline(t time.Time) error {
	return &net.OpError{Op: "set", Net: "pipe", Source: nil, Addr: nil, Err: errors.New("deadline not supported")}
}

func (s *stream) SetWriteDeadline(t time.Time) error {
	return &net.OpError{Op: "set", Net: "pipe", Source: nil, Addr: nil, Err: errors.New("deadline not supported")}
}

func (s *stream) transport() {
	defer s.teardown()

	bufsize := 256
	buf := new(bytes.Buffer)
	timer := time.NewTimer(0)
	if !timer.Stop() {
		select {
		case <-timer.C:
		default:
		}
	}

	defer timer.Stop()

	// 向通道中写入 buf 中的数据
	drainBuf := func() error {
		if buf.Len() > 0 {
			_, err := s.write.Write(buf.Bytes())
			if err != nil {
				return err
			}
			buf.Reset()
		}
		return nil
	}

	// 向 buf 中写入 msg
	deliverOrWait := func(o *transportObject) error {
		// 已经缓存的 + 计划缓存的
		buffered := len(o.msg) + buf.Len()
		// 停止计时器
		if !timer.Stop() {
			// 如果停止计时器失败，就消耗掉计时器信号
			select {
			case <-timer.C:
			default:
			}
		}

		// 计算需要延迟的时间，重新设置计时器
		delay := o.arrivalTime.Sub(time.Now())
		if delay >= 0 {
			timer.Reset(delay)
		} else {
			timer.Reset(0)
		}

		// 根据 buf 的尺寸，判断应该刷新缓存/缓存数据 ？
		if buffered >= bufsize {
			select {
			case <-timer.C:
			case <-s.reset:
				select {
				case s.reset <- struct{}{}:
				default:
				}
				return ErrReset
			}
			if err := drainBuf(); err != nil {
				return err
			}
			_, err := s.write.Write(o.msg)
			if err != nil {
				return err
			}
		} else {
			buf.Write(o.msg)
		}
		return nil
	}

	for {
		select {
		case <-s.reset:
			s.writeErr = ErrReset
			return
		default:
		}

		select {
		case <-s.reset:
		case <-s.close:
			if err := drainBuf(); err != nil {
				s.resetWith(err)
				return
			}
			s.writeErr = s.write.Close()
			if s.writeErr == nil {
				s.writeErr = ErrClosed
			}
			return

		case o := <-s.toDeliver: // 写缓存
			if err := deliverOrWait(o); err != nil {
				s.resetWith(err)
				return
			}
		case <-timer.C: // 刷新缓存
			if err := drainBuf(); err != nil {
				s.resetWith(err)
				return
			}
		}
	}
}

func (s *stream) resetWith(err error) {
	s.write.CloseWithError(err)
	s.read.CloseWithError(err)
	s.writeErr = err
}

func (s *stream) teardown() {
	s.conn.removeStream(s)
	close(s.closed)

	s.conn.net.notifyAll(func(n inet.Notifiee) {
		n.ClosedStream(s.conn.net, s)
	})
}
