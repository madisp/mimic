package main

import (
	"fmt"
	"github.com/madisp/mimic/rtsp"
)

func printUnit(unit []byte) error {
	//	fmt.Println("got NAL unit, len:", len(unit))
	return nil
}

func main() {
	//path := "10sec_raw.h264"
	//fmt.Println("Reading", path, "as h264")
	//if err := rtsp.Read(path, printUnit); err != nil {
	//	fmt.Printf("Reading h264 stream failed:\n\t%s\n", err)
	//}
	fmt.Println("RTSP server running on port 5554")
	rtsp.Serve(5554)
}
