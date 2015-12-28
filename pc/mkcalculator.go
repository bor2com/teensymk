package main

import (
	"fmt"
	pb "github.com/dibrov4bor/teensymk/pc-autogen/proto"
	"github.com/golang/protobuf/proto"
	"github.com/tarm/serial"
	"log"
)

type serialPort struct {
	port   *serial.Port
	buffer []byte
}

func openSerialPort(config *serial.Config) (*serialPort, error) {
	port, err := serial.OpenPort(config)
	if err != nil {
		return nil, fmt.Errorf("failed to open serial port: %s", err)
	}
	result := &serialPort{
		port:   port,
		buffer: make([]byte, 0, 10),
	}
	return result, nil
}

func (sp *serialPort) Marshal(message proto.Message) error {
	buf, err := proto.Marshal(message)
	if err != nil {
		return fmt.Errorf("failed to marshal protobuf: %s", err)
	}
	if _, err := sp.port.Write(proto.EncodeVarint(uint64(len(buf)))); err != nil {
		return fmt.Errorf("failed to write message length to serial port: %s", err)
	}
	if _, err := sp.port.Write(buf); err != nil {
		return fmt.Errorf("failed to write message to serial port: %s", err)
	}
	return nil
}

func (sp *serialPort) Unmarshal(message proto.Message) error {
	var headerLength, totalLength int
	for {
		var messageLength uint64
		messageLength, headerLength = proto.DecodeVarint(sp.buffer)
		totalLength = headerLength + int(messageLength)
		if totalLength != 0 && totalLength <= len(sp.buffer) {
			// Got the entire response message in the buffer.
			break
		}

		// Grow buffer capacity if necessary.
		if totalLength > cap(sp.buffer) {
			grownBuffer := make([]byte, len(sp.buffer), totalLength)
			copy(grownBuffer, sp.buffer)
			sp.buffer = grownBuffer
		}

		// Read from serial port.
		consumed, err := sp.port.Read(sp.buffer[len(sp.buffer):cap(sp.buffer)])
		if err != nil {
			return fmt.Errorf("failed to read from the serial stream: %s", err)
		}
		sp.buffer = sp.buffer[:len(sp.buffer)+consumed]
	}
	if err := proto.Unmarshal(sp.buffer[headerLength:totalLength], message); err != nil {
		return fmt.Errorf("failed to unmarshall the response: %s", err)
	}
	// Skip parsed bytes in the input buffer.
	copy(sp.buffer, sp.buffer[totalLength:])
	sp.buffer = sp.buffer[:len(sp.buffer)-totalLength]
	return nil
}

func (sp *serialPort) Close() {
	sp.port.Close()
}

func main() {
	port, err := openSerialPort(&serial.Config{Name: "/dev/ttyACM0", Baud: 115200})
	if err != nil {
		log.Fatal(err)
	}
	defer port.Close()

	requests := []*pb.Request{
		{
			One: proto.Int32(17),
			Two: proto.Int32(7),
		},
		{
			One: proto.Int32(-19),
			Two: proto.Int32(19),
		},
		{
			One: proto.Int32(1789392012),
			Two: proto.Int32(-54184243),
		},
		{
			One: proto.Int32(2147483647),
			Two: proto.Int32(-2147483648),
		},
	}

	for _, request := range requests {
		if err := port.Marshal(request); err != nil {
			log.Fatal(err)
		}
		response := &pb.Response{}
		if err := port.Unmarshal(response); err != nil {
			log.Fatal(err)
		}
		fmt.Println("Request:")
		fmt.Print(proto.MarshalTextString(request))
		fmt.Println("Response:")
		fmt.Println(proto.MarshalTextString(response))
	}
}
