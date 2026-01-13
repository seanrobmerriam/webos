package main

import (
	"fmt"
	"log"
	"time"

	"webos/pkg/protocol"
)

func main() {
	fmt.Println("WebOS Protocol Demo")
	fmt.Println("===================")
	fmt.Println()

	// 1. Creating and encoding messages
	fmt.Println("1. Creating and encoding messages...")

	connectMsg := protocol.NewMessage(
		protocol.OpcodeConnect,
		[]byte(`{"clientId":"web-client-123","version":"1.0"}`),
	)
	encoded, err := connectMsg.Encode()
	if err != nil {
		log.Fatal(err)
	}
	fmt.Printf("   Connect message encoded: %d bytes\n", len(encoded))

	textMsg := protocol.NewMessage(
		protocol.OpcodeDisplay,
		[]byte("Hello, WebOS!"),
	)
	encoded, err = textMsg.Encode()
	if err != nil {
		log.Fatal(err)
	}
	fmt.Printf("   Text message encoded: %d bytes\n", len(encoded))

	// Large binary message
	binaryData := make([]byte, 256)
	for i := range binaryData {
		binaryData[i] = byte(i)
	}
	binaryMsg := protocol.NewMessage(protocol.OpcodeFileSystem, binaryData)
	encoded, err = binaryMsg.Encode()
	if err != nil {
		log.Fatal(err)
	}
	fmt.Printf("   Binary message encoded: %d bytes\n", len(encoded))
	fmt.Println()

	// 2. Decoding messages
	fmt.Println("2. Decoding messages...")

	// Decode connect message
	var decoded protocol.Message
	if err := decoded.Decode(encoded); err != nil {
		log.Fatal(err)
	}
	fmt.Printf("   Decoded: Message{Opcode: %s, PayloadSize: %d}\n",
		decoded.Opcode, len(decoded.Payload))
	fmt.Printf("   Payload: %s\n", string(decoded.Payload))
	fmt.Println()

	// 3. Testing error handling
	fmt.Println("3. Testing error handling...")

	// Invalid magic
	invalidData := make([]byte, 20)
	var invalidMsg protocol.Message
	err = invalidMsg.Decode(invalidData)
	if err == protocol.ErrInvalidMagic {
		fmt.Printf("   ✓ Invalid magic error: %v\n", err)
	}

	// Truncated message
	truncatedData := []byte("WEBS\x01\x01" + string(make([]byte, 5)))
	var truncatedMsg protocol.Message
	err = truncatedMsg.Decode(truncatedData)
	if err == protocol.ErrBufferTooSmall {
		fmt.Printf("   ✓ Truncated message error: %v\n", err)
	}
	fmt.Println()

	// 4. Available opcodes
	fmt.Println("4. Available opcodes:")
	for i := protocol.OpcodeDisplay; i <= protocol.OpcodeError; i++ {
		fmt.Printf("   %d: %s\n", i, i)
	}
	fmt.Println()

	// 5. Timing information
	fmt.Println("5. Timing information:")
	start := time.Now()
	for i := 0; i < 10000; i++ {
		msg := protocol.NewMessage(protocol.OpcodeDisplay, []byte("test"))
		msg.Encode()
	}
	elapsed := time.Since(start)
	fmt.Printf("   10,000 encode operations: %v\n", elapsed)
	fmt.Printf("   Average per operation: %v\n", elapsed/10000)
	fmt.Println()

	// 6. Testing payload size limits
	fmt.Println("6. Testing payload size limits...")
	maxPayload := make([]byte, 16*1024*1024) // 16 MB
	maxMsg := protocol.NewMessage(protocol.OpcodeDisplay, maxPayload)
	_, err = maxMsg.Encode()
	if err != nil {
		fmt.Printf("   ✗ Max payload encoding failed: %v\n", err)
	} else {
		fmt.Printf("   ✓ Max payload (16 MB) encoded successfully\n")
	}

	// Too large
	tooLargePayload := make([]byte, 17*1024*1024) // 17 MB
	tooLargeMsg := protocol.NewMessage(protocol.OpcodeDisplay, tooLargePayload)
	_, err = tooLargeMsg.Encode()
	if err == protocol.ErrPayloadTooLarge {
		fmt.Printf("   ✓ Payload too large correctly rejected\n")
	}
	fmt.Println()

	// 7. Codec demonstration
	fmt.Println("7. Codec demonstration...")
	buf := make([]byte, 256)
	codec := protocol.NewCodec(buf)
	codec.WriteUint32(12345)
	codec.WriteUint64(9876543210)
	codec.WriteBytes([]byte("test string"))
	codec.Reset()

	val32, _ := codec.ReadUint32()
	val64, _ := codec.ReadUint64()
	valBytes, _ := codec.ReadBytes(11)
	fmt.Printf("   Read uint32: %d\n", val32)
	fmt.Printf("   Read uint64: %d\n", val64)
	fmt.Printf("   Read bytes: %s\n", string(valBytes))
	fmt.Println()

	fmt.Println("Demo completed successfully!")
}
