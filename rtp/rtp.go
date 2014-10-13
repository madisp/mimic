package rtp

import (
	"fmt"
	"github.com/madisp/mimic/h264"
	"math/rand"
	"net"
	"os"
	"strconv"
	"strings"
	"time"
)

var sessions map[string]*Session

type Session struct {
	ServerPort  string
	ClientPort  string
	SessionType string // like RTP/AVP
	SessionMode string // like unicast
	Id          string

	conn     net.Conn
	start    time.Time
	sequence uint16
}

type packet struct {
	/*
			   0                   1                   2                   3
		       0 1 2 3 4 5 6 7 8 9 0 1 2 3 4 5 6 7 8 9 0 1 2 3 4 5 6 7 8 9 0 1
		      +-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+
		      |V=2|P|X|  CC   |M|     PT      |       sequence number         |
		      +-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+
		      |                           timestamp                           |
		      +-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+
		      |           synchronization source (SSRC) identifier            |
		      +=+=+=+=+=+=+=+=+=+=+=+=+=+=+=+=+=+=+=+=+=+=+=+=+=+=+=+=+=+=+=+=+
		      |            contributing source (CSRC) identifiers             |
		      |                             ....                              |
		      +-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+
	*/
	version uint8  // 2
	p       uint8  // 0
	x       uint8  // 0
	cc      uint8  // 0
	m       uint8  // 0
	pt      uint8  // h264?
	seq     uint16 // seq
	ts      uint32 // clock, 90khz, in usec?
	ssrc    uint32 // wat?
	payload []byte // nalu with header removed
}

func (p packet) bytes() (ret []byte) {
	ret = make([]byte, 12)
	ret[0] = ((p.version & 0x03) << 6) + (p.p << 5) + (p.x << 4) + (p.cc & 0x0f)
	ret[1] = ((p.m << 7) & 0x80) + (p.pt & 0x7f)
	ret[2] = uint8((p.seq >> 8) & 0xff)
	ret[3] = uint8(p.seq & 0xff)
	ret[4] = uint8((p.ts >> 24) & 0xff)
	ret[5] = uint8((p.ts >> 16) & 0xff)
	ret[6] = uint8((p.ts >> 8) & 0xff)
	ret[7] = uint8((p.ts) & 0xff)
	ret[8] = uint8((p.ssrc >> 24) & 0xff)
	ret[9] = uint8((p.ssrc >> 16) & 0xff)
	ret[10] = uint8((p.ssrc >> 8) & 0xff)
	ret[11] = uint8((p.ssrc) & 0xff)
	ret = append(ret, p.payload...)
	return
}

func init() {
	sessions = map[string]*Session{}
}

func (self *Session) newPacket() (p packet) {
	p.version = 2
	p.p = 0
	p.x = 0
	p.cc = 0
	p.m = 0
	p.pt = 99
	p.seq = self.sequence
	self.sequence = self.sequence + 1
	p.ts = 0
	p.ssrc = 0
	return
}

func NewSession(transportDesc string, remoteAddr net.Addr) (s *Session) {
	s = &Session{}

	// deserialize input data
	parts := strings.Split(transportDesc, ";")
	s.SessionType = parts[0]
	s.SessionMode = parts[1]
	paramsMap := map[string]string{}
	for _, v := range parts[2:] {
		pair := strings.Split(v, "=")
		paramsMap[pair[0]] = pair[1]
	}
	s.ClientPort = strings.Split(paramsMap["client_port"], "-")[0]

	// pick a random udp port
	ip := strings.Split(remoteAddr.String(), ":")[0]
	conn, err := net.Dial("udp", fmt.Sprintf("%s:%s", ip, s.ClientPort))
	if err != nil {
		fmt.Println(err)
		return
	}
	s.ServerPort = strings.Split(conn.LocalAddr().String(), ":")[1]
	s.conn = conn
	s.Id = strconv.Itoa(rand.Int())
	s.start = time.Now()
	sessions[s.Id] = s
	return
}

func Play(sessId string) {
	fmt.Println("Playing session ", sessId)
	sess := sessions[sessId]
	go h264.Scan(os.Stdin, func(data []byte) error {
		// remove start zeroes
		for data[0] == 0x00 {
			data = data[1:]
		}
		// remove start one
		data = data[1:]

		ts := time.Now().UnixNano() - sess.start.UnixNano()

		if len(data) < 1024 {
			pack := sess.newPacket()
			pack.ts = uint32(ts / 11111)
			pack.payload = data
			if _, err := sess.conn.Write(pack.bytes()); err != nil {
				fmt.Println(err)
				return err
			}
		} else {
			off := 0
			// packetize in FU-A
			nalNri := (data[0] >> 5) & 0x03
			nalType := data[0] & 0x1f
			// eat nal-hdr
			data = data[1:]

			fuHdr := make([]byte, 2)
			fuHdr[0] = (nalNri << 5) + 28

			for off < len(data) {
				sz := len(data) - off
				if sz > 1024 {
					sz = 1024
				}
				fuHdr[1] = nalType
				if off == 0 {
					// fmt.Println("FU-A start flag set")
					fuHdr[1] = fuHdr[1] | (1 << 7)
				}
				if off+sz == len(data) {
					// fmt.Println("FU-A end flag set")
					fuHdr[1] = fuHdr[1] | (1 << 6)
					if off == 0 {
						panic("FU-A S+E flags both set")
					}
				}
				pack := sess.newPacket()
				pack.ts = uint32(ts / 11111)
				// fmt.Println("Sending packet", fuHdr, pack)
				// actual payload
				pack.payload = append(fuHdr, data[off:off+sz]...)

				if _, err := sess.conn.Write(pack.bytes()); err != nil {
					fmt.Println(err)
					return err
				}
				off += sz
			}
		}
		// fmt.Printf("NALU sent, %d bytes, hdr=%d\n", len(data), data[0])
		return nil
	})
}

func Destroy(sessId string) {
	fmt.Println("Destroy session ", sessId)
	s := sessions[sessId]
	s.conn.Close()
	sessions[sessId] = nil
}
