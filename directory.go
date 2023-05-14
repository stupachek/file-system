package main

import (
	"encoding/json"
	"errors"
)

type Entry struct {
	Name  string `json:"name"`
	Inode int64  `json:"inode"`
}

var ErrFileIsNotDir error = errors.New("file is not directory")
var ErrFileNotFound error = errors.New("file is not found")

func (f *FileSystem) AllocateDirectory() (Inode, error) {
	// allocate inode
	// set file type to directory
	// return
	inode, err := f.AllocateInode()
	if err != nil {
		return Inode{}, err
	}
	inode.fileType = DIRECTORY
	err = f.WriteDirectory(&inode, []Entry{})
	return inode, err
}

func (f *FileSystem) AddFile(dir *Inode, name string, file *Inode) error {
	if dir.fileType != DIRECTORY {
		return ErrFileIsNotDir
	}
	entries, err := f.ReadDirectory(dir)
	if err != nil {
		return err
	}
	for _, entry := range entries {
		if entry.Name == name {
			return errors.New("file already exists")
		}
	}
	entries = append(entries, Entry{
		Name:  name,
		Inode: file.id,
	})
	err = f.WriteDirectory(dir, entries)
	if err != nil {
		return err
	}
	file.linkCount++
	err = f.WriteInode(file)
	return err
}

func (f *FileSystem) RemoveFile(dir *Inode, name string) (Inode, error) {
	if dir.fileType != DIRECTORY {
		return Inode{}, ErrFileIsNotDir
	}
	entries, err := f.ReadDirectory(dir)
	if err != nil {
		return Inode{}, err
	}
	file := Inode{}
	var remove bool
	for i, entry := range entries {
		if entry.Name == name {
			file, err = f.ReadInode(entry.Inode)
			if err != nil {
				return Inode{}, err
			}
			entries = append(entries[:i], entries[i+1:]...)
			remove = true
		}
	}
	if !remove {
		return Inode{}, ErrFileNotFound
	}
	err = f.WriteDirectory(dir, entries)
	if err != nil {
		return Inode{}, err
	}
	file.linkCount--
	err = f.WriteInode(&file)
	return file, err
}

func (f *FileSystem) ReadDirectory(dir *Inode) ([]Entry, error) {
	if dir.fileType != DIRECTORY {
		return nil, ErrFileIsNotDir
	}
	buffer := make([]byte, dir.Size)
	_, err := f.Read(dir, 0, buffer)
	if err != nil {
		return []Entry{}, err
	}
	var entry []Entry
	err = json.Unmarshal(buffer, &entry)
	if err != nil {
		return []Entry{}, err
	}
	return entry, nil
}

func (f *FileSystem) WriteDirectory(dir *Inode, entry []Entry) error {
	if dir.fileType != DIRECTORY {
		return ErrFileIsNotDir
	}
	data, err := json.Marshal(entry)
	if err != nil {
		return err
	}
	err = f.Truncate(dir, 0)
	if err != nil {
		return err
	}
	_, err = f.Write(dir, 0, data)
	return err
}
