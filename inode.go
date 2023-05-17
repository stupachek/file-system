package main

import (
	"encoding/binary"
	"fmt"
	"io"
	"os"
)

type FileType byte

const (
	DIRECTORY    FileType = 0
	REGULAR      FileType = 1
	SYMLINK      FileType = 2
	DIRECT_LINKS          = 16
)

type Inode struct {
	id        int64
	fileType  FileType
	linkCount int64
	Size      int64
	Blocks    [DIRECT_LINKS]int64
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

	return nil
}

func (f *FileSystem) ReadInode(inode int64) (Inode, error) {
	// find the location of inode
	// and read it
	location := f.Superblock.InodesOffset + inode*INODE_SIZE
	f.File.Seek(location, io.SeekStart)
	i := Inode{
		id: inode,
	}
	err := i.Read(f.File)
	return i, err
}

func (f *FileSystem) WriteInode(inode *Inode) error {
	location := f.Superblock.InodesOffset + inode.id*INODE_SIZE
	f.File.Seek(location, io.SeekStart)
	err := inode.Write(f.File)
	return err
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

func (f *FileSystem) DeallocateInode(file *Inode) error {
	err := f.Truncate(file, 0)
	if err != nil {
		return err
	}
	return f.SetInodeBitmapOffset(file.id, FREE)
}
