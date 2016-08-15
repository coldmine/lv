package main

import (
	_ "bytes"
	"encoding/binary"
	"fmt"
	"log"
	"os"
)

type Atom struct {
	length int
	typ    string
	// data []byte
}

func main() {
	f, err := os.Open("m4x4_prores422hq.mov")
	if err != nil {
		log.Fatal(err)
	}
	defer f.Close()
	sizeByte := make([]byte, 4)
	_, err = f.Read(sizeByte)
	if err != nil {
		log.Fatal(err)
	}
	n32 := binary.BigEndian.Uint32(sizeByte)
	n := int(n32)
	fmt.Println(n)

	typeByte := make([]byte, 4)
	_, err = f.Read(typeByte)
	if err != nil {
		log.Fatal(err)
	}
	t := string(typeByte)
	fmt.Println(t)
}
