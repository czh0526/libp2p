package stream_muxer

import (
	crand "crypto/rand"
	"fmt"
	"io"
	mrand "math/rand"
	"net"
	"reflect"
	"runtime"
	"runtime/debug"
	"testing"

	smux "github.com/libp2p/go-stream-muxer"
)

func checkErr(t *testing.T, err error) {
	if err != nil {
		debug.PrintStack()
		t.Fatal(err)
	}
}

var randomness []byte

func init() {
	randomness = make([]byte, 1<<20)
	if _, err := crand.Read(randomness); err != nil {
		panic(err)
	}
}

func randBuf(size int) []byte {
	n := len(randomness) - size
	if size < 1 {
		panic(fmt.Errorf("requested too larrge buffer (%d). max is %d", size, len(randomness)))
	}
	start := mrand.Intn(n)
	return randomness[start : start+size]
}

func GoServe(t *testing.T, tr smux.Transport, l net.Listener) (done func()) {
	closed := make(chan struct{}, 1)

	go func() {
		for {
			cl, err := l.Accept()
			if err != nil {
				select {
				case <-closed:
					return
				default:
					checkErr(t, err)
				}
			}

			fmt.Println("accepted connection")
			conn, err := tr.NewConn(cl, true)
			checkErr(t, err)
			go func() {
				for {
					stream, err := conn.AcceptStream()
					if err != nil {
						break
					}
					go echoStream(stream)
				}
			}()
		}
	}()

	return func() {
		closed <- struct{}{}
	}
}

type LogWriter struct {
	W io.Writer
}

func (lw *LogWriter) Write(buf []byte) (int, error) {
	if testing.Verbose() {
		fmt.Printf("logwriter: writing %d bytes \n", len(buf))
	}
	return lw.W.Write(buf)
}

func echoStream(s smux.Stream) {
	defer s.Close()
	fmt.Println("accepted stream")
	io.Copy(&LogWriter{s}, s)
}

func SubtestSimpleWrite(t *testing.T, tr smux.Transport) {
	l, err := net.Listen("tcp", "localhost:0")
	checkErr(t, err)
	fmt.Printf("listening at %s \n", l.Addr().String())
	done := GoServe(t, tr, l)
	defer done()

	fmt.Printf("dialing to %s \n", l.Addr().String())
	ncl, err := net.Dial("tcp", l.Addr().String())
	checkErr(t, err)
	defer ncl.Close()

	fmt.Printf("【wrapping conn】: smux.Transport(%T) -> (net.Conn ==> smux.Conn)\n", tr)
	cl, err := tr.NewConn(ncl, false)
	checkErr(t, err)
	defer cl.Close()

	go cl.AcceptStream()

	fmt.Println("【Creating stream】: smux.Conn ==> smux.Stream")
	s1, err := cl.OpenStream()
	checkErr(t, err)
	defer s1.Close()

	buf1 := randBuf(4096)
	fmt.Printf("writing %d bytes to stream \n", len(buf1))
	_, err = s1.Write(buf1)
	checkErr(t, err)

	buf2 := make([]byte, len(buf1))
	fmt.Printf("reading %d bytes from tream (echoed) \n", len(buf2))
	_, err = s1.Read(buf2)
	checkErr(t, err)

	if string(buf2) != string(buf1) {
		t.Errorf("buf1 and buf2 not equal.")
	}
	fmt.Println("done")
}

func SubtestDummy(t *testing.T, tr smux.Transport) {
	fmt.Println("SubtestDummy")
}

type TransportTest func(t *testing.T, tr smux.Transport)

var Subtests = []TransportTest{
	//SubtestDummy,
	SubtestSimpleWrite,
}

func getFunctionName(i interface{}) string {
	return runtime.FuncForPC(reflect.ValueOf(i).Pointer()).Name()
}

func SubtestAll(t *testing.T, tr smux.Transport) {
	for _, f := range Subtests {
		t.Run(getFunctionName(f), func(t *testing.T) {
			f(t, tr)
		})
	}
}
