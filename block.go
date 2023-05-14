package main

import (
	"encoding/binary"
	"fmt"
	"io"
)

type Block int64

func (f *FileSystem) AllocateBlock() (Block, error) {
	block, err := f.FindFreeBlock()
	if err != nil {
		return -1, err
	}
	err = f.SetBlockBitmapOffset(block, USED)
	if err != nil {
		return -1, err
	}
	err = f.ClearBlock(block)
	if err != nil {
		return -1, err
	}
	return block, nil
}

func (f *FileSystem) ClearBlock(block Block) error {
	location := f.Superblock.BlocksOffset + int64(block)*BLOCK_SIZE
	f.File.Seek(location, io.SeekStart)
	bytes := make([]byte, BLOCK_SIZE)
	_, err := f.File.Write(bytes)
	return err
}

func (f *FileSystem) FindFreeBlock() (Block, error) {
	start := f.Superblock.BlockBitmapOffset
	f.File.Seek(start, io.SeekStart)
	var bbyte int64 = 0
	for ; bbyte < f.Superblock.BlockCount/8; bbyte++ {
		var word byte
		err := binary.Read(f.File, binary.BigEndian, &word)
		if err != nil {
			return -1, err
		}

		if word != 0xff {
			for bbit := int64(0); bbit < 8; bbit++ {
				if (word & 0b0000_0001) == 0 {
					return Block(bbyte*8 + bbit), nil
				}
				word >>= 1
			}
		}
	}

	return -1, fmt.Errorf("out of memory")
}

func (f *FileSystem) SetBlockBitmapOffset(block Block, status int) error {
	byte_position := f.Superblock.BlockBitmapOffset + int64(block)/8
	f.File.Seek(byte_position, io.SeekStart)
	var bbyte byte
	err := binary.Read(f.File, binary.BigEndian, &bbyte)
	if err != nil {
		return err
	}
	bbit := block % 8
	if status == FREE {
		var mask byte = ^(1 << bbit) //  bbit = 3; 1 << bbit = 0b0000_1000
		//                              ^(1 << bbit) = 0b1111_0111
		bbyte = bbyte & mask
	} else {
		bbyte = bbyte | (1 << bbit) // bbit = 3; 1 << bbit = 0b0000_1000
		// 0bxxxxxxxxx
		// 0bxxxxx1xxx
	}
	f.File.Seek(byte_position, io.SeekStart)
	err = binary.Write(f.File, binary.BigEndian, bbyte)
	if err != nil {
		return err
	}
	return nil
}
