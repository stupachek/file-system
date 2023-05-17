package main

import (
	"errors"
	"path"
	"strings"
)

var ErrFileIsNotRegular error = errors.New("file is not regular")
var ErrDirIsNotEmpty error = errors.New("directory is not empty")
var ErrDelDot error = errors.New("can't delete \".\" or \"..\"")
var ErrSym error = errors.New("too many levels of symbolic links")

type Stat struct {
	inode   int64
	ftype   FileType
	size    int64
	links   int64
	symlink string
}

func (f *FileSystem) Create(dir int64, name string, ftype FileType) (int64, error) {
	directory, err := f.ReadInode(dir)
	if err != nil {
		return -1, err
	}
	if directory.fileType != DIRECTORY {
		return -1, ErrFileIsNotDir
	}
	file := Inode{}
	if ftype == REGULAR {
		file, err = f.AllocateInode()
		if err != nil {
			return -1, err
		}
		err = f.WriteInode(&file)
		if err != nil {
			return -1, err
		}

	} else {
		file, err = f.AllocateDirectory()
		if err != nil {
			return -1, err
		}
		err = f.AddFile(&file, ".", &file)
		if err != nil {
			return -1, err
		}
		err = f.AddFile(&file, "..", &directory)
		if err != nil {
			return -1, err
		}
	}

	err = f.AddFile(&directory, name, &file)
	if err != nil {
		return -1, err
	}
	return file.id, nil
}

func (f *FileSystem) List(dir int64) ([]Entry, error) {
	// read inode
	inode, err := f.ReadInode(dir)
	if err != nil {
		return nil, err
	}
	// read directory entries
	entry, err := f.ReadDirectory(&inode)
	return entry, err
}

func (f *FileSystem) Lookup(dir int64, name string) (int64, error) {
	// read inode
	// read directory entries
	entry, err := f.List(dir)
	if err != nil {
		return -1, err
	}
	// find the file with the given name and return the inode
	for _, file := range entry {
		if file.Name == name {
			return file.Inode, nil
		}
	}
	return -1, ErrFileNotFound
}

func (f *FileSystem) ReadFile(file int64, offset int64, buffer []byte) (int64, error) {
	// read the inode
	inode, err := f.ReadInode(file)
	if err != nil {
		return -1, err
	}
	// check that it's file
	if inode.fileType != REGULAR {
		return -1, ErrFileIsNotRegular
	}
	// read data
	return f.Read(&inode, offset, buffer)

}

func (f *FileSystem) WriteFile(file int64, offset int64, buffer []byte) (int64, error) {
	// read the inode
	inode, err := f.ReadInode(file)
	if err != nil {
		return -1, err
	}
	// check that it's file
	if inode.fileType != REGULAR {
		return -1, ErrFileIsNotRegular
	}
	// write data
	return f.Write(&inode, offset, buffer)
}

func (f *FileSystem) LinkFile(dir int64, name string, file int64) error {
	// read the dir
	// check that it's directory
	directory, err := f.ReadInode(dir)
	if err != nil {
		return err
	}
	if directory.fileType != DIRECTORY {
		return ErrFileIsNotDir
	}
	// read the file, check that it's file
	fileForLink, err := f.ReadInode(file)
	if err != nil {
		return err
	}
	if fileForLink.fileType != REGULAR {
		return ErrFileIsNotRegular
	}
	// add the file to the directory
	err = f.AddFile(&directory, name, &fileForLink)
	return err
}

func (f *FileSystem) UnlinkFile(dir int64, name string) error {
	// forbid "." and ".."
	if name == "." || name == ".." {
		return ErrDelDot
	}
	// read the dir
	// check that it's directory
	directory, err := f.ReadInode(dir)
	if err != nil {
		return err
	}
	if directory.fileType != DIRECTORY {
		return ErrFileIsNotDir
	}
	in, err := f.Lookup(dir, name)
	if err != nil {
		return err
	}
	file, err := f.ReadInode(in)
	if err != nil {
		return err
	}
	if file.fileType == DIRECTORY {
		return ErrFileIsNotRegular
	}
	// remove the file
	file, err = f.RemoveFile(&directory, name)
	if err != nil {
		return err
	}
	// deallocate the inode, it the counter is 0
	if file.linkCount == 0 {
		err = f.DeallocateInode(&file)
		if err != nil {
			return err
		}
	}

	return nil
}

func (f *FileSystem) Stat(file int64) (Stat, error) {
	// read inode
	inode, err := f.ReadInode(file)
	if err != nil {
		return Stat{}, err
	}
	// fill struct
	symlink := ""
	if inode.fileType == SYMLINK {
		symlink, err = f.ReadFileStr(inode.id)
		if err != nil {
			return Stat{}, err
		}
	}
	return Stat{
		inode:   inode.id,
		ftype:   inode.fileType,
		size:    inode.Size,
		links:   inode.linkCount,
		symlink: symlink,
	}, err
}

func (f *FileSystem) ResolvePathRecursive(pwd int64, path string, depth int) (int64, error) {

	if depth > 10 {
		return -1, ErrSym
	}
	directories := strings.Split(path, "/")
	cwd := pwd
	if path == "" {
		return cwd, nil
	}
	if path[0] == '/' {
		directories = directories[1:]
		cwd = f.Superblock.Root
	}
	if path[len(path)-1] == '/' {
		directories = directories[:len(directories)-1]
	}
	for _, dirName := range directories {
		dir, err := f.Lookup(cwd, dirName)
		if err != nil {
			return -1, err
		}
		dir, err = f.ResolveSymlink(cwd, dir, depth)
		if err != nil {
			return -1, err
		}
		cwd = dir
	}
	return cwd, nil
}

func (f *FileSystem) ResolveSymlink(pwd int64, inode int64, depth int) (int64, error) {
	file, err := f.ReadInode(inode)
	if err != nil {
		return -1, err
	}
	if file.fileType != SYMLINK {
		return inode, nil
	}
	path, err := f.ReadFileStr(inode)
	if err != nil {
		return -1, err
	}
	return f.ResolvePathRecursive(pwd, path, depth+1)
}

func (f *FileSystem) ResolvePath(pwd int64, path string) (int64, error) {
	return f.ResolvePathRecursive(pwd, path, 0)
}

func (f *FileSystem) ResolveParent(pwd int64, fullPath string) (string, int64, error) {
	dir, file := path.Split(fullPath)
	inode, err := f.ResolvePath(pwd, dir)
	return file, inode, err
}

func (f *FileSystem) RemoveDir(dir int64, name string) error {
	directory, err := f.ReadInode(dir)
	if err != nil {
		return err
	}
	if directory.fileType != DIRECTORY {
		return ErrFileIsNotDir
	}
	in, err := f.Lookup(dir, name)
	if err != nil {
		return err
	}
	file, err := f.ReadInode(in)
	if err != nil {
		return err
	}
	if file.fileType != DIRECTORY {
		return ErrFileIsNotDir
	}
	entry, err := f.List(file.id)
	if err != nil {
		return err
	}
	if len(entry) != 2 {
		return ErrDirIsNotEmpty
	}
	_, err = f.RemoveFile(&file, ".")
	if err != nil {
		return err
	}
	_, err = f.RemoveFile(&file, "..")
	if err != nil {
		return err
	}
	file, err = f.RemoveFile(&directory, name)
	if err != nil {
		return err
	}
	// deallocate the inode, it the counter is 0
	if file.linkCount == 0 {
		err = f.DeallocateInode(&file)
		if err != nil {
			return err
		}
	}

	return nil
}

func (f *FileSystem) SymlinkFile(dir int64, name string, symlink string) error {
	// create file in "dir" in the "name"
	directory, err := f.ReadInode(dir)
	if err != nil {
		return err
	}
	if directory.fileType != DIRECTORY {
		return ErrFileIsNotDir
	}
	file, err := f.AllocateInode()
	if err != nil {
		return err
	}
	// make it symlink
	file.fileType = SYMLINK
	err = f.WriteInode(&file)
	if err != nil {
		return err
	}
	// write the "symlink" to the file
	_, err = f.Write(&file, 0, []byte(symlink))
	if err != nil {
		return err
	}
	err = f.AddFile(&directory, name, &file)
	if err != nil {
		return err
	}
	return nil
}

func (f *FileSystem) ReadFileStr(inode int64) (string, error) {
	file, err := f.ReadInode(inode)
	if err != nil {
		return "", err
	}
	buff := make([]byte, file.Size)
	_, err = f.Read(&file, 0, buff)
	if err != nil {
		return "", err
	}
	return string(buff), nil

}
