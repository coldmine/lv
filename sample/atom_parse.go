package main

import (
	"encoding/binary"
	"fmt"
	"log"
	"os"
)

type Atom struct {
	size int
	typ  string
	// data []byte
}

func main() {
	atoms := make([]Atom, 0)

	pth := "m4x4_prores422hq.mov"
	f, err := os.Open(pth)
	if err != nil {
		log.Fatal("Could not open file: ", err)
	}
	defer f.Close()

	fi, err := os.Stat(pth)
	if err != nil {
		log.Fatal("Could not get file info: ", err)
	}

	for {
		sizeByte := make([]byte, 4)
		_, err = f.Read(sizeByte)
		if err != nil {
			log.Fatal("Could not read atom size: ", err)
		}
		n32 := binary.BigEndian.Uint32(sizeByte)
		n := int(n32)

		typeByte := make([]byte, 4)
		_, err = f.Read(typeByte)
		if err != nil {
			log.Fatal("Could not read atom type: ", err)
		}
		t := string(typeByte)
		atoms = append(atoms, Atom{size: n, typ: t})

		// seek next atom
		off, err := f.Seek(int64(n-8), 1)
		if off == fi.Size() {
			// reached at EOF
			break
		}
		if err != nil {
			log.Fatal("Could not seek to next atom: ", err)
		}
	}

	fmt.Println(atoms)
}
