package main

import (
	"errors"
	"fmt"
)

var ErrUnknownFS error = errors.New("unknown file descriptor")

func (f *FileSystem) CreateCmd(pwd int64, path string) error {
	_, err := f.Create(pwd, path, REGULAR)
	return err
}

func (f *FileSystem) LinkCmd(pwd int64, from string, to string) error {
	inode, err := f.Lookup(pwd, from)
	if err != nil {
		return err
	}
	err = f.LinkFile(pwd, to, inode)
	return err
}

func (f *FileSystem) UnlinkCmd(pwd int64, name string) error {
	err := f.UnlinkFile(pwd, name)
	return err
}

func (f *FileSystem) TruncateCmd(pwd int64, name string, size int64) error {
	inodeId, err := f.Lookup(pwd, name)
	if err != nil {
		return err
	}
	inode, err := f.ReadInode(inodeId)
	if err != nil {
		return err
	}
	err = f.Truncate(&inode, size)
	return err
}

func (f *FileSystem) StatCmd(pwd int64, name string) (Stat, error) {
	inodeId, err := f.Lookup(pwd, name)
	if err != nil {
		return Stat{}, err
	}
	return f.Stat(inodeId)
}

func (f *FileSystem) OpenCmd(pwd int64, name string) (Fkey, error) {
	// read inode
	inodeId, err := f.Lookup(pwd, name)
	if err != nil {
		return "", err
	}
	// allocate new Fkey
	key := fmt.Sprintf("%v", f.Session.counter)
	f.Session.counter++
	// add Fd to sessions
	f.Session.fds[Fkey(key)] = &Fd{
		inode:    inodeId,
		location: 0,
	}
	return Fkey(key), nil
}

func (f *FileSystem) WriteCmd(fd Fkey, data string) error {
	// get inode from sessions
	fileDesc, ok := f.Session.fds[fd]
	if !ok {
		return ErrUnknownFS
	}
	inodeId := fileDesc.inode
	inode, err := f.ReadInode(inodeId)
	if err != nil {
		return err
	}
	// write data to the file
	n, err := f.Write(&inode, fileDesc.location, []byte(data))
	if err != nil {
		return err
	}
	// update location
	fileDesc.location += n
	return nil
}

func (f *FileSystem) ReadCmd(fd Fkey, length int64) (string, error) {
	// get inode from sessions
	fileDesc, ok := f.Session.fds[fd]
	if !ok {
		return "", ErrUnknownFS
	}
	inodeId := fileDesc.inode
	inode, err := f.ReadInode(inodeId)
	if err != nil {
		return "", err
	}
	// read data from file
	buff := make([]byte, length)
	n, err := f.Read(&inode, fileDesc.location, buff)
	if err != nil {
		return "", err
	}
	// update location
	fileDesc.location += n
	return string(buff), nil
}

func (f *FileSystem) SeekCmd(fd Fkey, offset int64) error {
	// get inode from sessions
	fileDesc, ok := f.Session.fds[fd]
	if !ok {
		return ErrUnknownFS
	}
	// set the location to the offset
	fileDesc.location = offset
	return nil
}

func (f *FileSystem) CloseCmd(fd Fkey) error {
	// remove entry from the sessions
	if _, ok := f.Session.fds[fd]; !ok {
		return ErrUnknownFS
	}
	delete(f.Session.fds, fd)
	return nil
}
