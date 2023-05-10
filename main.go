package main

import (
	"encoding/binary"
	"fmt"
	"io"
	"log"
	"os"
)

const (
	SUPERBLOCK_SIZE = 64
	INODE_SIZE      = 1
	BLOCK_SIZE      = 64
	FREE            = 0
	USED            = 1
)

type FileSystem struct {
	File       *os.File
	Superblock Superblock
}

type Superblock struct {
	Size              int64
	InodeBitmapOffset int64
	BlockBitmapOffset int64
	InodesOffset      int64
	BlocksOffset      int64
	InodeCount        int64
	BlockCount        int64
}

func NewFileSystem(count int64, path string) (FileSystem, error) {
	var inode_count int64 = count
	var block_count int64 = count
	bitmap_size := inode_count/8 + block_count/8
	system_size := SUPERBLOCK_SIZE + inode_count*INODE_SIZE + block_count*BLOCK_SIZE + bitmap_size
	f, err := os.OpenFile(path, os.O_RDWR|os.O_CREATE|os.O_TRUNC, 0644)
	if err != nil {
		return FileSystem{}, err
	}
	err = f.Truncate(system_size)
	if err != nil {
		return FileSystem{}, err
	}
	var inode_bitmap int64 = SUPERBLOCK_SIZE
	var block_bitmap int64 = inode_bitmap + inode_count/8
	var inode int64 = block_bitmap + block_count/8
	var block int64 = inode + inode_count*INODE_SIZE
	superblock := Superblock{
		Size:              system_size,
		InodeBitmapOffset: inode_bitmap,
		BlockBitmapOffset: block_bitmap,
		InodesOffset:      inode,
		BlocksOffset:      block,
		InodeCount:        inode_count,
		BlockCount:        block_count,
	}

	return FileSystem{
		File:       f,
		Superblock: superblock,
	}, nil
}

func (s *Superblock) Write(file *os.File) error {
	err := binary.Write(file, binary.BigEndian, s.Size)
	if err != nil {
		return err
	}
	err = binary.Write(file, binary.BigEndian, s.InodeBitmapOffset)
	if err != nil {
		return err
	}
	err = binary.Write(file, binary.BigEndian, s.BlockBitmapOffset)
	if err != nil {
		return err
	}
	err = binary.Write(file, binary.BigEndian, s.InodesOffset)
	if err != nil {
		return err
	}
	err = binary.Write(file, binary.BigEndian, s.BlocksOffset)
	if err != nil {
		return err
	}
	err = binary.Write(file, binary.BigEndian, s.InodeCount)
	if err != nil {
		return err
	}
	err = binary.Write(file, binary.BigEndian, s.BlockCount)
	if err != nil {
		return err
	}
	return nil
}

func (f *FileSystem) Close() error {
	f.File.Seek(0, io.SeekStart)
	err := f.Superblock.Write(f.File)
	if err != nil {
		return err
	}
	return f.File.Close()
}

func (s *Superblock) Read(file *os.File) error {
	err := binary.Read(file, binary.BigEndian, &s.Size)
	if err != nil {
		return err
	}
	err = binary.Read(file, binary.BigEndian, &s.InodeBitmapOffset)
	if err != nil {
		return err
	}
	err = binary.Read(file, binary.BigEndian, &s.BlockBitmapOffset)
	if err != nil {
		return err
	}
	err = binary.Read(file, binary.BigEndian, &s.InodesOffset)
	if err != nil {
		return err
	}
	err = binary.Read(file, binary.BigEndian, &s.BlocksOffset)
	if err != nil {
		return err
	}
	err = binary.Read(file, binary.BigEndian, &s.InodeCount)
	if err != nil {
		return err
	}
	err = binary.Read(file, binary.BigEndian, &s.BlockCount)
	if err != nil {
		return err
	}
	return nil
}

func (f *FileSystem) AllocateInode() (Inode, error) {
	id, err := f.FindFreeInode()
	if err != nil {
		return Inode{}, err
	}

	err = f.SetInodeBitmapOffset(id, USED)
	if err != nil {
		return Inode{}, err
	}

	return Inode{
		id:        id,
		fileType:  REGULAR,
		linkCount: 0,
		Size:      0,
	}, nil
}

func (f *FileSystem) SetInodeBitmapOffset(inode int64, status int) error {
	byte_position := f.Superblock.InodeBitmapOffset + inode/8
	f.File.Seek(byte_position, io.SeekStart)
	var bbyte byte
	err := binary.Read(f.File, binary.BigEndian, &bbyte)
	if err != nil {
		return err
	}
	bbit := inode % 8
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

func (f *FileSystem) FindFreeInode() (int64, error) {
	start := f.Superblock.InodeBitmapOffset
	f.File.Seek(start, io.SeekStart)
	var bbyte int64 = 0
	for ; bbyte < f.Superblock.InodeCount/8; bbyte++ {
		var word byte
		err := binary.Read(f.File, binary.BigEndian, &word)
		if err != nil {
			return -1, err
		}

		if word != 0xff {
			for bbit := int64(0); bbit < 8; bbit++ {
				if (word & 0b0000_0001) == 0 {
					return bbyte*8 + bbit, nil
				}
				word >>= 1
			}
		}
	}

	return -1, fmt.Errorf("out of memory")
}

func main() {
	f, err := NewFileSystem(16, "qwe")
	if err != nil {
		log.Fatal(err)
	}
	for i := 0; i < 18; i++ {
		inode, err := f.AllocateInode()
		fmt.Printf("%+v %s\n", inode, err)
	}
	f.Close()
}
