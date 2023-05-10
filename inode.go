package main

import (
	"encoding/binary"
	"os"
)

const (
	DIRECTORY    = 0
	REGULAR      = 1
	DIRECT_LINKS = 16
)

type Inode struct {
	id            int64
	fileType      byte
	linkCount     int64
	Size          int64
	Blocks        [DIRECT_LINKS]int64
	IndirectBlock int64
}

func (i *Inode) Write(file *os.File) error {
	err := binary.Write(file, binary.BigEndian, i.fileType)
	if err != nil {
		return err
	}
	err = binary.Write(file, binary.BigEndian, i.linkCount)
	if err != nil {
		return err
	}
	err = binary.Write(file, binary.BigEndian, i.Size)
	if err != nil {
		return err
	}

	for j := 0; j < DIRECT_LINKS; j++ {
		err = binary.Write(file, binary.BigEndian, i.Blocks[j])
		if err != nil {
			return err
		}
	}

	err = binary.Write(file, binary.BigEndian, i.IndirectBlock)
	if err != nil {
		return err
	}

	return nil
}

func (i *Inode) Read(file *os.File) error {
	err := binary.Read(file, binary.BigEndian, &i.fileType)
	if err != nil {
		return err
	}
	err = binary.Read(file, binary.BigEndian, &i.linkCount)
	if err != nil {
		return err
	}
	err = binary.Read(file, binary.BigEndian, &i.Size)
	if err != nil {
		return err
	}

	for j := 0; j < DIRECT_LINKS; j++ {
		err = binary.Read(file, binary.BigEndian, &i.Blocks[j])
		if err != nil {
			return err
		}
	}

	err = binary.Read(file, binary.BigEndian, &i.IndirectBlock)
	if err != nil {
		return err
	}

	return nil
}
