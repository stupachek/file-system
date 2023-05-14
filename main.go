package main

import (
	"encoding/binary"
	"io"
	"os"
)

const (
	SUPERBLOCK_SIZE = 64
	INODE_SIZE      = 1 + 2*8 + 8*DIRECT_LINKS + 8
	BLOCK_SIZE      = 1024
	FREE            = 0
	USED            = 1
)

type FileSystem struct {
	File       *os.File
	Superblock Superblock
	Session    Session
}

type Superblock struct {
	Size              int64
	InodeBitmapOffset int64
	BlockBitmapOffset int64
	InodesOffset      int64
	BlocksOffset      int64
	InodeCount        int64
	BlockCount        int64
	Root              int64
}

type Fd struct {
	inode    int64
	location int64
}

type Fkey string

type Session struct {
	pwd     int64
	fds     map[Fkey]*Fd
	counter int64
}

func NewFileSystem(count int64, path string) (FileSystem, error) {
	count = UpDivision(count, 8) * 8
	var inode_count int64 = count
	var block_count int64 = 10 + 10*count
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
		Root:              0,
	}
	fileS := FileSystem{
		File:       f,
		Superblock: superblock,
	}
	root, err := fileS.AllocateDirectory()
	if err != nil {
		return FileSystem{}, err
	}
	err = fileS.AddFile(&root, ".", &root)
	if err != nil {
		return FileSystem{}, err
	}
	err = fileS.AddFile(&root, "..", &root)
	if err != nil {
		return FileSystem{}, err
	}
	fileS.Superblock.Root = root.id
	fileS.Session.pwd = root.id
	fileS.Session.fds = make(map[Fkey]*Fd)
	return fileS, nil
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
	err = binary.Write(file, binary.BigEndian, s.Root)
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
	err = binary.Read(file, binary.BigEndian, &s.Root)
	if err != nil {
		return err
	}
	return nil
}

func main() {
	f, err := NewFileSystem(128, "fs")
	if err != nil {
		panic(err)
	}

	repl := NewRepl(&f)
	repl.Start()
	f.Close()
}
