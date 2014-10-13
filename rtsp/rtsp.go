package rtsp

import (
	"fmt"
	"github.com/madisp/mimic/rtp"
	"io"
	"net"
	"strconv"
	"strings"
)

func Serve(port int) error {
	sock, err := net.Listen("tcp", fmt.Sprintf(":%d", port))
	if err != nil {
		return err
	}
	for {
		conn, err := sock.Accept()
		if err != nil {
			fmt.Println(err)
			continue
		}
		go handleConnection(conn)
	}
	return nil
}

type request struct {
	method  string
	uri     string
	version string
	headers map[string]string

	out net.Conn
}

func (self request) respond(headers map[string]string, data []byte) {
	//TODO status? body?
	fmt.Fprintf(self.out, "RTSP/1.0 200 OK\r\n")
	fmt.Printf("RTSP/1.0 200 OK\r\n")

	// prepare headers
	headers["CSeq"] = self.headers["CSeq"]
	if len(data) > 0 {
		headers["Content-Length"] = strconv.Itoa(len(data))
	}

	for k, v := range headers {
		fmt.Fprintf(self.out, "%s: %s\r\n", k, v)
		fmt.Printf("%s: %s\r\n", k, v)
	}
	fmt.Fprintf(self.out, "\r\n")
	if len(data) > 0 {
		self.out.Write(data)
		fmt.Println(string(data))
	}
	fmt.Printf("\r\n")
}

func handleRequest(r request) {
	fmt.Println("---", r.method, r.uri, r.version, "---")
	fmt.Println(r.headers)
	if r.method == "OPTIONS" {
		r.respond(map[string]string{"Public": "DESCRIBE, SETUP, TEARDOWN, PLAY"}, nil)
	} else if r.method == "DESCRIBE" {
		data := []byte("v=0\no=madis 1 1 127.0.0.1\ns=mimic cast\nt=0 0\nm=video 0 RTP/AVP 99\na=rtpmap:99 H264/90000\na=fmtp:99 profile-level-id=64000d;packetization-mode=1")
		r.respond(map[string]string{"Content-Type": "application/sdp"}, data)
	} else if r.method == "SETUP" {
		sess := rtp.NewSession(r.headers["Transport"], r.out.RemoteAddr())
		transportStr := fmt.Sprintf("%s;%s;client_port=%s;server_port=%s", sess.SessionType, sess.SessionMode, sess.ClientPort, sess.ServerPort)
		r.respond(map[string]string{"Session": sess.Id, "Transport": transportStr}, nil)
	} else if r.method == "PLAY" {
		rtp.Play(r.headers["Session"])
		r.respond(map[string]string{"Session": r.headers["Session"]}, nil)
	} else if r.method == "TEARDOWN" {
		rtp.Destroy(r.headers["Session"])
		r.respond(map[string]string{}, nil)
	}
}

func handleConnection(conn net.Conn) {
	fmt.Printf("New connection from %s\n", conn.RemoteAddr())
	buf := make([]byte, 4096)
	for {
		input := ""
		// read until CRLFCRLF
		for len(input) == 0 || input[len(input)-4:len(input)] != "\r\n\r\n" {
			if n, err := conn.Read(buf); err != nil {
				if err != io.EOF {
					fmt.Println(err)
					return
				} else {
					// EOF
					fmt.Printf("Connection to %s closed\n", conn.RemoteAddr())
					conn.Close()
					return
				}
			} else {
				input += string(buf[0:n])
			}
		}
		// trim trailing CRLFCRLF
		for strings.HasSuffix(input, "\r\n") {
			input = strings.TrimSuffix(input, "\r\n")
		}
		lines := strings.Split(input, "\r\n")

		reqLine := strings.Split(lines[0], " ")
		headers := make(map[string]string)
		for i := 1; i < len(lines); i++ {
			pair := strings.SplitN(lines[i], ":", 2)
			headers[strings.TrimSpace(pair[0])] = strings.TrimSpace(pair[1])
		}
		//TODO read body?
		handleRequest(request{reqLine[0], reqLine[1], reqLine[2], headers, conn})
	}
}
